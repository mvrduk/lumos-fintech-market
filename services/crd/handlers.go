package main

import (
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
)

func addCardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var req AddCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad json", http.StatusBadRequest)
		return
	}

	if req.UserID == 0 || len(req.CardNumber) < 4 {
		http.Error(w, "invalid data", http.StatusBadRequest)
		return
	}

	if !luhnCheck(req.CardNumber) {
		http.Error(w, "invalid card", http.StatusBadRequest)
		return
	}

	last4 := req.CardNumber[len(req.CardNumber)-4:]

	brand := "unknown"

	if strings.HasPrefix(req.CardNumber, "4") {
		brand = "visa"
	} else if strings.HasPrefix(req.CardNumber, "5") {
		brand = "masercard"
	}

	token := "tok_" + last4 + "_" + strconv.Itoa(req.UserID)

	var cardID int

	err := db.QueryRow(
		`INSERT INTO cards (user_id, last4, brand, card_token)
				VALUES ($1, $2, $3, $4)
				RETURNING id
				`, req.UserID, last4, brand, token).Scan(&cardID)
	if err != nil {
		logger.Error("error inserting card into db", zap.Error(err))
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	response := AddCardResponse{
		CardID: cardID,
		Last4:  last4,
		Brand:  brand,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func luhnCheck(cardNumber string) bool {
	sum := 0

	parity := len(cardNumber) % 2

	for i, d := range cardNumber {
		digit := int(d - '0')
		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return sum%10 == 0
}
