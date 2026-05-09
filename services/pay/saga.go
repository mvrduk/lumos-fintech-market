package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func executePaymentSaga(payment Payment) error {

	walletURL := os.Getenv("WALLET_SERVICE_URL")
	if walletURL == "" {
		walletURL = "http://wlt-service:8084"
	}

	client := http.Client{Timeout: 3 * time.Second}

	fromWalletID, err := getWalletID(payment.FromUserID)
	if err != nil {
		return fmt.Errorf("failed to get sender wallet: %w", err)
	}

	toWalletID, err := getWalletID(payment.ToUserID)
	if err != nil {
		return fmt.Errorf("failed to get receiver wallet: %w", err)
	}

	err = callWallet(client, walletURL+"/api/wlt/debit", map[string]interface{}{
		"wallet_id": fromWalletID,
		"amount":    payment.Amount,
	})

	if err != nil {
		return fmt.Errorf("debit failed: %w", err)
	}

	err = callWallet(client, walletURL+"/api/wlt/credit", map[string]interface{}{
		"wallet_id": toWalletID,
		"amount":    payment.Amount,
	})

	if err != nil {
		_ = callWallet(client, walletURL+"/api/wlt/credit", map[string]interface{}{
			"wallet_id": fromWalletID,
			"amount":    payment.Amount,
		})

		return fmt.Errorf("credit failed, rollback executed: %w", err)
	}

	return nil
}

func getWalletID(userID int) (int, error) {

	walletURL := os.Getenv("WALLET_SERVICE_URL")
	if walletURL == "" {
		walletURL = "http://wlt-service:8084"
	}

	resp, err := http.Get(fmt.Sprintf("%s/api/wlt/wallets?user_id=%d", walletURL, userID))
	if err != nil {
		return 0, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	var wallets []struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&wallets); err != nil {
		return 0, err
	}

	if len(wallets) == 0 {
		return 0, fmt.Errorf("no wallet for user %d", userID)
	}

	return wallets[0].ID, nil
}

func callWallet(client http.Client, url string, body map[string]interface{}) error {

	return retry(3, 200*time.Millisecond, func() error {

		jsonData, _ := json.Marshal(body)

		resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("wallet returned %d", resp.StatusCode)
		}

		return nil
	})
}
