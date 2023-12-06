package chaincode

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type Patient struct {
	contractapi.Contract
}

type PatientWallet struct {
	HealthRecords []HealthRecord `json:"healthRecords"`
}

type HealthRecord struct {
	Description string `json:"description"`
	CreatedDate int64  `json:"createDate"`
	Date        int64  `json:"date"`
	EntityName  string `json:"entityName"`
	RecordType  string `json:"type"`
}

func (c *Patient) AddDataToWallet(ctx contractapi.TransactionContextInterface,
	content, socialSecurityNumber, entityName, recordType string, date int64) error {

	compositeKey, err := createCompositeKey(ctx, socialSecurityNumber)
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	patientWallet, err := getCurrentPatientWallet(ctx, compositeKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	newRecord := HealthRecord{
		Description: content,
		CreatedDate: time.Now().Unix(),
		Date:        date,
		EntityName:  entityName,
		RecordType:  recordType,
	}

	patientWallet.HealthRecords = append(patientWallet.HealthRecords, newRecord)

	if err := updateWallet(ctx, *patientWallet, compositeKey); err != nil {
		return fmt.Errorf("failed to update the wallet: %v", err)
	}

	return nil
}

// Vamos obter todo o histórico do utente.
func (c *Patient) GetMedicalHistory(ctx contractapi.TransactionContextInterface,
	socialSecurityNumber string) ([]HealthRecord, error) {

	compositeKey, err := createCompositeKey(ctx, socialSecurityNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key: %v", err)
	}

	patientWallet, err := getCurrentPatientWallet(ctx, compositeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	return patientWallet.HealthRecords, nil
}

func createCompositeKey(ctx contractapi.TransactionContextInterface, key string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("HealthRecord", []string{"socialSecurityNumber", key})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key: %v", err)
	}
	return compositeKey, nil
}

func getCurrentPatientWallet(ctx contractapi.TransactionContextInterface,
	key string) (*PatientWallet, error) {

	walletBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read patient wallet: %v", err)
	}

	var patientWallet PatientWallet

	if walletBytes != nil {
		err = json.Unmarshal(walletBytes, &patientWallet)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal patient wallet: %v", err)
		}
	} else {
		patientWallet.HealthRecords = []HealthRecord{}
	}

	return &patientWallet, nil
}

func updateWallet(ctx contractapi.TransactionContextInterface, patientWallet PatientWallet, key string) error {
	updatedWalletBytes, err := json.Marshal(patientWallet)
	if err != nil {
		return fmt.Errorf("failed to marshal updated patient wallet: %v", err)
	}

	err = ctx.GetStub().PutState(key, updatedWalletBytes)
	if err != nil {
		return fmt.Errorf("failed to update patient wallet: %v", err)
	}

	return nil
}

func GenerateUniqueID(socialSecurityNumber string) string {
	hasher := sha256.New()
	hasher.Write([]byte(socialSecurityNumber)) // Write the data into the hasher
	hashInBytes := hasher.Sum(nil)             // Sum(nil) calculates the hash and returns bytes

	hashString := hex.EncodeToString(hashInBytes) // Convert bytes to hexadecimal string

	return hashString
}
