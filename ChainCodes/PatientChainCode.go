package chaincodes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	chaincode, err := contractapi.NewChaincode(&PatientChaincode{})
	if err != nil {
		fmt.Printf("Error creating PatientChaincode: %v", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting PatientChaincode: %v", err)
	}
}

type PatientChaincode struct {
	contractapi.Contract
}

func (c *PatientChaincode) AddDataToWallet(ctx contractapi.TransactionContextInterface, recordTypeID int, content string) error {

	patientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get patient ID: %v", err)
	}

	walletBytes, err := ctx.GetStub().GetState(patientID)
	if err != nil {
		return fmt.Errorf("failed to read patient wallet: %v", err)
	}

	var patientWallet PatientWallet
	if walletBytes == nil {
		patientWallet = PatientWallet{
			OwnerID:       patientID,
			HealthRecords: []HealthRecord{},
			Consents:      []HealthRecordConsent{},
		}
	} else {
		err = json.Unmarshal(walletBytes, &patientWallet)
		if err != nil {
			return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
		}
	}

	newRecord := HealthRecord{
		RecordID:     GenerateUniqueID(),
		RecordTypeID: recordTypeID,
		Content:      content,
	}

	patientWallet.HealthRecords = append(patientWallet.HealthRecords, newRecord)

	// Update the patient wallet on the ledger
	updatedWalletBytes, err := json.Marshal(patientWallet)
	if err != nil {
		return fmt.Errorf("failed to marshal updated patient wallet: %v", err)
	}

	err = ctx.GetStub().PutState(patientID, updatedWalletBytes)
	if err != nil {
		return fmt.Errorf("failed to update patient wallet: %v", err)
	}

	return nil
}

// GrantConsent grants consent to an organization to access the patient's data
func (c *PatientChaincode) GrantConsent(ctx contractapi.TransactionContextInterface, organizationID string) error {
	// Get the patient's identity (medical ID)
	patientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get patient ID: %v", err)
	}

	// Retrieve existing patient wallet
	walletBytes, err := ctx.GetStub().GetState(patientID)
	if err != nil {
		return fmt.Errorf("failed to read patient wallet: %v", err)
	}

	if walletBytes == nil {
		return fmt.Errorf("patient wallet not found")
	}

	var patientWallet PatientWallet

	err = json.Unmarshal(walletBytes, &patientWallet)
	if err != nil {
		return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	// Grant consent to the organization
	// patientWallet.ConsentMap[organizationID] = true

	// Update the patient wallet on the ledger
	updatedWalletBytes, err := json.Marshal(patientWallet)
	if err != nil {
		return fmt.Errorf("failed to marshal updated patient wallet: %v", err)
	}

	err = ctx.GetStub().PutState(patientID, updatedWalletBytes)
	if err != nil {
		return fmt.Errorf("failed to update patient wallet: %v", err)
	}

	return nil
}

// GetMedicalHistory retrieves the patient's medical history
func (c *PatientChaincode) GetMedicalHistory(ctx contractapi.TransactionContextInterface) (*PatientWallet, error) {

	patientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return nil, fmt.Errorf("failed to get patient ID: %v", err)
	}

	walletBytes, err := ctx.GetStub().GetState(patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to read patient wallet: %v", err)
	}

	if walletBytes == nil {
		return nil, fmt.Errorf("patient wallet not found")
	}

	var patientWallet PatientWallet
	err = json.Unmarshal(walletBytes, &patientWallet)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	return &patientWallet, nil
}

func GenerateUniqueID() string {
	hasher := sha256.New()

	hashInBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashInBytes)

	return hashString
}
