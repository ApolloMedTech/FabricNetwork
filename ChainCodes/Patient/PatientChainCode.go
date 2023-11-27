package patient

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type PatientChaincode struct {
	contractapi.Contract
}

type HealthRecord struct {
	Description string `json:"description"`
}

func (c *PatientChaincode) AddDataToWallet(ctx contractapi.TransactionContextInterface,
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
	}

	newRecord := HealthRecord {
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
func (c *PatientChaincode) GetMedicalHistory(ctx contractapi.TransactionContextInterface,
											 socialSecurityNumber string) (*PatientWallet, error) {
	
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

	var patientWallet PatientWallet
	err = json.Unmarshal(walletBytes, &patientWallet)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	return &patientWallet, nil
}


func GenerateUniqueID(socialSecurityNumber string) string {
	hasher := sha256.New()

	hashInBytes := hasher.Sum(socialSecurityNumber)
	hashString := hex.EncodeToString(hashInBytes)

	return hashString
}

// Método de start quando o chaincode leva deploy.
func main() {
	chaincode, err := contractapi.NewChaincode(&PatientChaincode{})
	if err != nil {
		fmt.Printf("Error creating PatientChaincode: %v", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting PatientChaincode: %v", err)
	}

	fmt.Printf("Se chegou aqui então correu bem e foi lançado corretamente")
}
