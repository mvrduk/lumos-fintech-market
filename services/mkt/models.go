package main

type Product struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price string `json:"price"`
}

type Order struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	ProductID int    `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Total     string `json:"total"`
	Status    string `json:"status"`
}

type CreateOrderRequest struct {
	UserID    int `json:"user_id"`
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type CreateOrderResponse struct {
	OrderID int    `json:"order_id"`
	Status  string `json:"status"`
	Total   string `json:"total"`
}
