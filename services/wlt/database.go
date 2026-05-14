package main

import (
	"database/sql"
	"fmt"
	"github.com/shopspring/decimal"

	_ "github.com/shopspring/decimal"
)

func CreateWallet(db *sql.DB, userID int, currency string) (*Wallet, error) {
	query := `INSERT INTO wallets (user_id, balance, currency) VALUES ($1, $2, $3) RETURNING id, user_id, balance, currency`

	w := &Wallet{}

	err := db.QueryRow(query, userID, 0, currency).Scan(&w.ID, &w.UserId, &w.Balance, &w.Currency)
	if err != nil {
		return nil, fmt.Errorf("error inserting in DB: %v", err)
	}

	return w, nil
}

func GetWallet(db *sql.DB, walletId int) (*Wallet, error) {
	query := `SELECT id, user_id, balance, currency FROM wallets WHERE id = $1`

	w := &Wallet{}
	err := db.QueryRow(query, walletId).Scan(&w.ID, &w.UserId, &w.Balance, &w.Currency)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, fmt.Errorf("error querying wallet: %v", err)
	}
	return w, nil
}

func GetWalletsByUserID(db *sql.DB, userID int) ([]*Wallet, error) {
	query := `SELECT id, user_id, balance, currency FROM wallets WHERE user_id = $1 ORDER BY currency ASC`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("Error querying wallets: %v", err)
	}

	defer rows.Close()

	var wallets []*Wallet

	for rows.Next() {
		w := &Wallet{}
		err := rows.Scan(&w.ID, &w.UserId, &w.Balance, &w.Currency)
		if err != nil {
			return nil, fmt.Errorf("error scanning wallet row: %v", err)
		}
		wallets = append(wallets, w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during iteration: %v", err)
	}

	return wallets, nil
}

func GetWalletByUserIDAndCurrency(db *sql.DB, userID int, currency string) (*Wallet, error) {
	query := `SELECT id, user_id, balance, currency FROM wallets WHERE user_id = $1 and currency = $2`
	w := &Wallet{}
	err := db.QueryRow(query, userID, currency).Scan(&w.ID, &w.UserId, &w.Balance, &w.Currency)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, fmt.Errorf("error querying wallet: %v", err)
	}

	return w, nil
}

func WalletExists(db *sql.DB, walletID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM wallets WHERE id = $1)`

	var exists bool
	err := db.QueryRow(query, walletID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking wallet existence: %v", err)
	}

	return exists, nil
}

func DebitToWallet(db *sql.DB, walletID int, amount decimal.Decimal, reason string) (*Transaction, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("Failed to begin transaction: %v", err)
	}

	var exists bool
	err = tx.QueryRow("SELECT EXIST(SELECT 1 FROM wallets WHERE id = $1)", walletID).Scan(&exists)
	if err != nil || !exists {
		tx.Rollback()
		return nil, fmt.Errorf("wallet not found")
	}

	query := `UPDATE wallets SET balance = balance + $1 where id = $2`
	_, err = tx.Exec(query, amount, walletID)

	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update balance: %v", err)
	}

	txnQuery := `INSERT INTO wallet_transactions (wallet_id, amount, type, reason, created_at) 
                       values ($1, $2, $3, $4, NOW())
                       RETURNING id, wallet_id, amount, type, reason, created_at`

	txn := &Transaction{}
	err = tx.QueryRow(txnQuery, walletID, amount, "debit", reason).
		Scan(&txn.ID, &txn.WalletID, &txn.Amount, &txn.Type, &txn.Reason, &txn.CreatedAt)

	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to record transaction: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to commit: %v", err)
	}

	return txn, nil
}

func CreditFromWallet(db *sql.DB, walletID int, amount decimal.Decimal, reason string) (*Transaction, error) {
	tx, err := db.Begin()
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("wallet not found")
	}

	var currentBalance decimal.Decimal
	query := `SELECT balance from wallets WHERE id=$1 FOR UPDATE`

	err = tx.QueryRow(query, walletID).Scan(&currentBalance)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("wallet not found")
	}

	if currentBalance.LessThan(amount) {
		tx.Rollback()
		return nil, fmt.Errorf("insufficient funds")
	}

	updateQuery := `UPDATE wallets SET balance = balance - $1 WHERE id = $2`
	_, err = tx.Exec(updateQuery, amount, walletID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update balance: %v", err)
	}

	txnQuery := `INSERT INTO wallet_transactions (wallet_id, amount, type, reason, created_at) 
				VALUES ($1, $2, $3, $4, NOW())
				RETURNING id, wallet_id, amount, type,reason, created_at`

	txn := &Transaction{}
	err = tx.QueryRow(txnQuery, walletID, amount, "credit", reason).
		Scan(&txn.ID, &txn.WalletID, &txn.Amount, &txn.Type, &txn.Reason, &txn.Reason, &txn.CreatedAt)

	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to record transaction: %v", err)
	}

	err = tx.Commit()

	if err != nil {
		return nil, fmt.Errorf("failed to commit: %v", err)
	}

	return txn, nil
}

func GetWalletTransactionHistory(db *sql.DB, walletID int, page int, limit int) ([]*Transaction, error) {
	offset := (page - 1) * limit

	query := `
           SELECT id, wallet_id, amount, type, reason, created_at
			FROM wallet_transactions
			WHERE wallet_id = $1
			ORDER BY created_at DESC 
			LIMIT $2 OFFSET $3`

	rows, err := db.Query(query, walletID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to query transactons: %v", err)
	}

	defer rows.Close()

	var transactions []*Transaction
	for rows.Next() {
		txn := Transaction{}
		err := rows.Scan(&txn.ID, &txn.WalletID, &txn.Amount, &txn.Type, &txn.Reason, &txn.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scantransaction: %v", err)
		}

		transactions = append(transactions, &txn)
	}

	return transactions, rows.Err()
}
