package patient

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/ApolloMedTech/FabricNetwork/tree/master/ChainCodes/wallet"
)

type PatientChaincode struct {
	contractapi.Contract
}

func (c *PatientChaincode) AddDataToWallet(ctx contractapi.TransactionContextInterface,
													 recordTypeID int, content string) error {

	walletBytes, err := ctx.GetStub().GetState(patientID)
    if err != nil {
        return fmt.Errorf("failed to read patient wallet: %v", err)
    }

    var patientWallet PatientWallet

    err = json.Unmarshal(walletBytes, &patientWallet)
    if err != nil {
        return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
    }

    newRecord := HealthRecord {
        RecordID:     GenerateUniqueID(),
        RecordTypeID: recordTypeID,
        Content:      content,
    }

    patientWallet.HealthRecords = append(patientWallet.HealthRecords, newRecord)

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

// Método para conceder permissão a uma determinada entidade e utilizador.
func (c *PatientChaincode) GrantConsent(ctx contractapi.TransactionContextInterface,
													 newConsent HealthRecordConsent) error {
	
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

	// Vamos adicionar um novo consentimento. 
	patientWallet.Consents = append(patientWallet.Consents, newConsent)

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

// Vamos obter todo o histórico do utente.
func (c *PatientChaincode) GetMedicalHistory(ctx contractapi.TransactionContextInterface, 
														patientID string) (*PatientWallet, error) {

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

	fmt.Printf("Se chegou aqui então correu bem e foi lançado corretamente.")
}
