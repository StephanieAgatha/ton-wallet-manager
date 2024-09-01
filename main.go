package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"os"
	"strings"
)

func readPhraseFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return strings.Fields(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}

func readReceiversFromFile(filename string) (map[string]string, error) {
	fmt.Printf("Note: This bot only sends to 4 addresses at the same time.")
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	receivers := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ",")
		if len(parts) == 2 {
			receivers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return receivers, nil
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	client := liteclient.NewConnectionPool()

	// Connect to mainnet
	configUrl := "https://ton.org/global.config.json"
	err := client.AddConnectionsFromConfigUrl(context.Background(), configUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to mainnet")
	}

	api := ton.NewAPIClient(client, ton.ProofCheckPolicyFast).WithRetry()

	//read seed from file
	words, err := readPhraseFromFile("phrase.txt")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read seed phrase from file")
	}

	//initialize high-load wallet
	w, err := wallet.FromSeed(api, words, wallet.V4R2)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize wallet")
	}

	log.Info().Str("address", w.WalletAddress().String()).Msg("Wallet address")

	block, err := api.CurrentMasterchainInfo(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get current masterchain info")
	}

	balance, err := w.GetBalance(context.Background(), block)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get balance")
	}

	//display balance
	log.Info().Str("balance", balance.String()).Msg("Total balance of sender's wallet")

	//read receivers
	receivers, err := readReceiversFromFile("receiver.txt")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read receivers from file")
	}

	if balance.Nano().Uint64() >= 3000000 {
		var messages []*wallet.Message
		//generate message for each destination
		for addrStr, amtStr := range receivers {
			addr := address.MustParseAddr(addrStr)
			messages = append(messages, &wallet.Message{
				Mode: wallet.PayGasSeparately + wallet.IgnoreErrors,
				InternalMessage: &tlb.InternalMessage{
					IHRDisabled: true,
					Bounce:      addr.IsBounceable(),
					DstAddr:     addr,
					Amount:      tlb.MustFromTON(amtStr),
				},
			})
		}

		log.Info().Msg("Sending transaction and waiting for confirmation...")

		txHash, err := w.SendManyWaitTxHash(context.Background(), messages)
		if err != nil {
			log.Fatal().Err(err).Msg("Transfer failed")
		}

		txHashStr := base64.URLEncoding.EncodeToString(txHash)
		log.Info().
			Str("hash", base64.StdEncoding.EncodeToString(txHash)).
			Str("explorer_link", "https://tonscan.org/tx/"+txHashStr).
			Msg("Transaction sent")

		//log individual transfers
		for _, msg := range messages {
			log.Info().
				Str("address", msg.InternalMessage.DstAddr.String()).
				Str("amount", msg.InternalMessage.Amount.String()).
				Msg("Transaction sent")
		}

		//display the new balance after transactions
		newBalance, err := w.GetBalance(context.Background(), block)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get new balance after transactions")
		} else {
			log.Info().Str("new_balance", newBalance.String()).Msg("New balance of sender's wallet after transactions")
		}
	} else {
		log.Warn().Str("balance", balance.String()).Msg("Not enough balance")
	}
}
