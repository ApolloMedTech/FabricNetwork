package chaincode

import (
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func createPatientWalletCompositeKey(ctx contractapi.TransactionContextInterface, patientID string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("PatientWallet", []string{"patientID", patientID})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key: %v", err)
	}
	return compositeKey, nil
}

func createRequestCompositeKey(ctx contractapi.TransactionContextInterface, patientID, healthcareProfessionalID, requestID string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("Requests", []string{"patientID", patientID, "healthcareProfessionalID", healthcareProfessionalID, "requestID", requestID})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key: %v", err)
	}
	return compositeKey, nil
}

func createAcessesCompositeKey(ctx contractapi.TransactionContextInterface, requestID string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("Acesses", []string{"requestID", requestID})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key: %v", err)
	}
	return compositeKey, nil
}
