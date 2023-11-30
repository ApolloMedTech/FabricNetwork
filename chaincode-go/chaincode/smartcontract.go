package chaincode

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type Patient struct {
	contractapi.Contract
}

type HealthRecord struct {
	Description string `json:"description"`
}

func (c *Patient) AddDataToWallet(ctx contractapi.TransactionContextInterface,
	content string, socialSecurityNumber string) error {

	compositeKey, err := ctx.GetStub().CreateCompositeKey("HealthRecord", []string{"socialSecurityNumber", socialSecurityNumber})
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	walletBytes, err := ctx.GetStub().GetState(compositeKey)
	if err != nil {
		return fmt.Errorf("failed to read patient wallet: %v", err)
	}

	var patientWallet struct {
		HealthRecords []HealthRecord `json:"healthRecords"`
	}

	if walletBytes != nil {
		err = json.Unmarshal(walletBytes, &patientWallet)
		if err != nil {
			return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
		}
	} else {
		// Initialize the HealthRecords slice if the wallet is empty
		patientWallet.HealthRecords = []HealthRecord{}
	}

	newRecord := HealthRecord{
		Description: content,
	}

	patientWallet.HealthRecords = append(patientWallet.HealthRecords, newRecord)

	updatedWalletBytes, err := json.Marshal(patientWallet)
	if err != nil {
		return fmt.Errorf("failed to marshal updated patient wallet: %v", err)
	}

	err = ctx.GetStub().PutState(compositeKey, updatedWalletBytes)
	if err != nil {
		return fmt.Errorf("failed to update patient wallet: %v", err)
	}

	return nil
}

// Vamos obter todo o histórico do utente.
func (c *Patient) GetMedicalHistory(ctx contractapi.TransactionContextInterface,
	socialSecurityNumber string) ([]HealthRecord, error) {

	compositeKey, err := ctx.GetStub().CreateCompositeKey("HealthRecord", []string{"socialSecurityNumber", socialSecurityNumber})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key: %v", err)
	}

	walletBytes, err := ctx.GetStub().GetState(compositeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read patient wallet: %v", err)
	}

	if walletBytes == nil {
		return nil, fmt.Errorf("patient wallet not found")
	}

	var patientWallet struct {
		HealthRecords []HealthRecord `json:"healthRecords"`
	}

	err = json.Unmarshal(walletBytes, &patientWallet)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	return patientWallet.HealthRecords, nil
}

func GenerateUniqueID(socialSecurityNumber string) string {
	hasher := sha256.New()
	hasher.Write([]byte(socialSecurityNumber)) // Write the data into the hasher
	hashInBytes := hasher.Sum(nil)             // Sum(nil) calculates the hash and returns bytes

	hashString := hex.EncodeToString(hashInBytes) // Convert bytes to hexadecimal string

	return hashString
}
