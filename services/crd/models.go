package main

type Card struct {
	ID     int    `json:"id"`
	UserID int    `json:"user_id"`
	Last4  string `json:"last4"`
	Brand  string `json:"brand"`
	Token  string `json:"token"`
}

type AddCardRequest struct {
	UserID     int    `json:"user_id"`
	CardNumber string `json:"card_number"`
	ExpMonth   int    `json:"exp_month"`
	ExpYear    int    `json:"exp_year"`
	CVV        string `json:"cvv"`
}

type AddCardResponse struct {
	CardID int    `json:"card_id"`
	Last4  string `json:"last4"`
	Brand  string `json:"brand"`
}
