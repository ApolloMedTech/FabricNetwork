package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type HealthContract struct {
	contractapi.Contract
}

func getMedicalHistory(ctx contractapi.TransactionContextInterface, patientID string) ([]HealthRecord, error) {

	var healthRecords = []HealthRecord{}

	// Injeto o ID da wallet e assim é mais rápido.
	queryString := fmt.Sprintf(`{
        "selector": {
			"resourceType": 3,
			"patientID": "%s"
        }
    }`, patientID)

	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)

	// Aqui não posso dar erro, tenho de fazer desta maneira
	if err != nil {
		return healthRecords, nil
	}

	defer queryResultsIterator.Close()

	for queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()

		var healthRecord HealthRecord

		if err != nil {
			return nil, fmt.Errorf("erro ao obter os dados do paciente: %v", err)
		}

		if err := json.Unmarshal(queryResponse.Value, &healthRecord); err != nil {
			return nil, fmt.Errorf("erro ao transformar os dados na wallet: %v", err)
		}

		healthRecords = append(healthRecords, healthRecord)
	}

	return healthRecords, nil
}

func getHealthRecordByID(ctx contractapi.TransactionContextInterface, patientID, recordID string) (*HealthRecord, error) {

	queryString := fmt.Sprintf(`{
        "selector": {
            "_id": "\u0000%s\u0000%s\u0000%s\u0000%s\u0000%s\u0000"
        }
    }`, "HealthRecords", "patientID", patientID, "recordID", recordID)

	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)

	// Aqui não posso dar erro, tenho de fazer desta maneira
	if err != nil {
		return &HealthRecord{}, nil
	}

	defer queryResultsIterator.Close()

	var healthRecord HealthRecord

	if queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()

		if err != nil {
			return nil, fmt.Errorf("erro ao obter os dados do paciente: %v", err)
		}

		if err := json.Unmarshal(queryResponse.Value, &healthRecord); err != nil {
			return nil, fmt.Errorf("erro ao transformar os dados na wallet: %v", err)
		}
	} else {
		healthRecord = HealthRecord{}
	}

	return &healthRecord, nil
}

func addAccess(ctx contractapi.TransactionContextInterface, requestID, patientID, patientName, healthcareProfessionalID, healthcareProfessional string, expirationDate int64) error {
	// Create a new access based on the approved request
	access := Access{
		ResourceType:             2,
		RequestID:                requestID,
		PatientID:                patientID,
		PatientName:              patientName,
		HealthcareProfessionalID: healthcareProfessionalID,
		HealthcareProfessional:   healthcareProfessional,
		CreatedDate:              time.Now().Unix(),
		ExpirationDate:           expirationDate, // or set the expiration date as needed
	}

	// Serialize the access object to JSON
	accessJSON, err := json.Marshal(access)
	if err != nil {
		return fmt.Errorf("failed to serialize access to JSON: %v", err)
	}

	// Generate composite key for the access
	compositeKey, err := createAcessesCompositeKey(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to create composite key for access: %v", err)
	}

	// Store the serialized access on the ledger
	err = ctx.GetStub().PutState(compositeKey, accessJSON)
	if err != nil {
		return fmt.Errorf("failed to store access on the ledger: %v", err)
	}

	return nil
}

func storeRequest(ctx contractapi.TransactionContextInterface, request Request) error {

	requestAlreadyExist := checkIfRequestAlreadyExist(ctx, request.PatientID, request.HealthcareProfessionalID, request.RequestID)

	if requestAlreadyExist {
		return fmt.Errorf("request already exist: %v")
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to serialize request to JSON: %v", err)
	}

	compositeKey, err := createRequestCompositeKey(ctx, request.PatientID, request.HealthcareProfessionalID, request.RequestID)

	if err != nil {
		return fmt.Errorf("failed to create composite key for request: %v", err)
	}

	err = ctx.GetStub().PutState(compositeKey, requestJSON)
	if err != nil {
		return fmt.Errorf("failed to store request on the ledger: %v", err)
	}

	return nil
}

func checkIfRequestAlreadyExist(ctx contractapi.TransactionContextInterface, patientID, healthcareProfessionalID, requestID string) bool {

	queryString := fmt.Sprintf(`{
        "selector": {
            "healthcareProfessionalID": "%s",
			"resourceType": 1,
			"status" : 0,
			"patientID" : %s,
			"requestID" : %s
        }
    }`, patientID, healthcareProfessionalID, requestID)

	return checkIfAnyDataAlreadyExist(ctx, queryString)
}

func checkIfHealthRecordAlreadyExist(ctx contractapi.TransactionContextInterface, recordID, patientID string) bool {

	queryString := fmt.Sprintf(`{
        "selector": {
            "_id": "\u0000%s\u0000%s\u0000%s\u0000%s\u0000"
        }
    }`, "HealthRecords", "patientID", patientID, recordID)

	return checkIfAnyDataAlreadyExist(ctx, queryString)
}

func checkIfAnyDataAlreadyExist(ctx contractapi.TransactionContextInterface, query string) bool {

	queryResultsIterator, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return false
	}
	defer queryResultsIterator.Close()

	return queryResultsIterator.HasNext()
}
