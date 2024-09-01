package main

import (
	"bufio"
	"context"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false, FormatMessage: func(i interface{}) string {
		return i.(string)
	}})
}

func readAddressFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var addresses []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		addresses = append(addresses, strings.TrimSpace(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return addresses, nil
}

func MassBalanceCheck(api *ton.APIClient, block *ton.BlockIDExt) error {
	addresses, err := readAddressFromFile("address.txt")
	if err != nil {
		return err
	}

	log.Info().Msg("Checking balances...")

	for _, addr := range addresses {
		parsedAddr, err := address.ParseAddr(addr)
		if err != nil {
			log.Error().Err(err).Str("address", addr).Msg("Failed to parse address")
			continue
		}

		account, err := api.GetAccount(context.Background(), block, parsedAddr)
		if err != nil {
			log.Error().Err(err).Str("address", addr).Msg("Failed to get account")
			continue
		}

		if account == nil || account.State == nil {
			log.Error().Str("address", addr).Msg("Has no balance")
			continue
		}

		balance := account.State.Balance
		log.Info().Msgf("address = %s, balance = %s", addr, balance.String())
	}
	return nil
}

func main() {
	client := liteclient.NewConnectionPool()

	configUrl := "https://tonutils.com/ls/free-mainnet-config.json"
	err := client.AddConnectionsFromConfigUrl(context.Background(), configUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to mainnet")
	}

	api := ton.NewAPIClient(client)

	block, err := api.CurrentMasterchainInfo(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get current masterchain info")
	}

	err = MassBalanceCheck(api, block)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to do mass balance check")
	}
}
