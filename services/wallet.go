package services

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func GetWallet() solana.PublicKey {
	// Replace this with your Phantom wallet's private key if you want to use the full wallet
	// Or just use the public address if you only need to receive funds
	phantomAddress := "79hjkpSwnJ4g7PJ7YYQfJRGEwHwWWUB7ziyve15fC4YC" // Replace this with your address
	pubKey, err := solana.PublicKeyFromBase58(phantomAddress)
	if err != nil {
		log.Fatalf("Failed to parse wallet address: %v", err)
	}
	return pubKey
}

func AttemptBuy(wallet solana.PublicKey, targetToken solana.PublicKey, amount float64) error {
	// Connect to Solana mainnet
	client := rpc.New(rpc.MainNetBeta_RPC)

	// Add these definitions before creating swap instruction
	ammId := solana.MustPublicKeyFromBase58("YOUR_AMM_ID_HERE")
	userSourceTokenAccount := wallet // This should be your SOL account
	poolSourceTokenAccount := solana.MustPublicKeyFromBase58("POOL_SOURCE_TOKEN_ACCOUNT")
	poolDestinationTokenAccount := solana.MustPublicKeyFromBase58("POOL_DESTINATION_TOKEN_ACCOUNT")
	userDestinationTokenAccount := solana.MustPublicKeyFromBase58("YOUR_TARGET_TOKEN_ACCOUNT")
	lpMint := solana.MustPublicKeyFromBase58("LP_MINT_ADDRESS")
	feeAccount := solana.MustPublicKeyFromBase58("FEE_ACCOUNT_ADDRESS")

	// Rest of the implementation remains the same
	balance := CheckBalance(client, wallet)
	if balance < amount {
		return fmt.Errorf("insufficient balance: %.2f SOL", balance)
	}

	// Create swap instruction
	programID := solana.MustPublicKeyFromBase58("SwaPpA9LAaLfeLi3a68M4DjnLqgtticKg6CnyNwgAC8")
	instruction := CreateSwapInstruction(
		programID,
		ammId,
		userSourceTokenAccount,
		poolSourceTokenAccount,
		poolDestinationTokenAccount,
		userDestinationTokenAccount,
		lpMint,
		feeAccount,
		wallet,
		uint64(amount*1e9),
		uint64(0),
	)

	// Get recent blockhash
	recentBlockhash, err := client.GetRecentBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Create and use the instruction in a transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recentBlockhash.Value.Blockhash, // Use the fetched blockhash
		solana.TransactionPayer(wallet),
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Send the transaction
	sig, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	log.Printf("Transaction sent: %s", sig.String())

	return nil
}

func CreateSwapInstruction(
	programID solana.PublicKey,
	ammId solana.PublicKey,
	userSourceTokenAccount solana.PublicKey,
	poolSourceTokenAccount solana.PublicKey,
	poolDestinationTokenAccount solana.PublicKey,
	userDestinationTokenAccount solana.PublicKey,
	lpMint solana.PublicKey,
	feeAccount solana.PublicKey,
	userAuthority solana.PublicKey,
	amountIn uint64,
	minAmountOut uint64,
) solana.Instruction {
	data := make([]byte, 10)
	data[0] = 9 // Swap instruction code
	binary.LittleEndian.PutUint64(data[1:], amountIn)
	data[9] = uint8(minAmountOut)

	accounts := solana.AccountMetaSlice{
		{PublicKey: ammId, IsSigner: false, IsWritable: true},
		{PublicKey: userAuthority, IsSigner: true, IsWritable: false},
		{PublicKey: userSourceTokenAccount, IsSigner: false, IsWritable: true},
		{PublicKey: poolSourceTokenAccount, IsSigner: false, IsWritable: true},
		{PublicKey: poolDestinationTokenAccount, IsSigner: false, IsWritable: true},
		{PublicKey: userDestinationTokenAccount, IsSigner: false, IsWritable: true},
		{PublicKey: lpMint, IsSigner: false, IsWritable: false},
		{PublicKey: feeAccount, IsSigner: false, IsWritable: true},
	}

	return solana.NewInstruction(programID, accounts, data)
}

func CheckBalance(client *rpc.Client, wallet solana.PublicKey) float64 {
	balance, err := client.GetBalance(
		context.Background(),
		wallet,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		log.Printf("Failed to get balance: %v", err)
		return 0
	}
	return float64(balance.Value) / 1e9 // Convert lamports to SOL
}
