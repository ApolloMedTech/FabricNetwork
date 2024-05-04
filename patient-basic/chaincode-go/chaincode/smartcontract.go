package chaincode

// patient contract.
import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type PatientContract struct {
	contractapi.Contract
}

type HealthRecord struct {
	Description              string `json:"description"`
	HealthCareProfessionalID string `json:"healthCareProfessionalID"`
	HealthCareProfessional   string `json:"healthCareProfessional"`
	CreatedDate              int64  `json:"createdDate"`
	EventDate                int64  `json:"eventDate"`
	Speciality               string `json:"speciality"`
	RecordType               string `json:"recordType"`
	Organization             string `json:"organization"`
}

// Estrutura de Request
type Request struct {
	RequestID                string `json:"requestID"`
	Description              string `json:"description"`
	Organization             string `json:"organization"`
	HealthCareProfessionalID string `json:"healthcareProfessionalID"`
	HealthCareProfessional   string `json:"healthcareProfessional"`
	PatientID                string `json:"patientID"`
	CreatedDate              int64  `json:"createdDate"`
	Status                   int    `json:"status"`
	StatusChangedDate        int64  `json:"statusChangedDate"`
}

type Access struct {
	RequestID                string `json:"requestID"`
	PatientID                string `json:"patientID"`
	HealthcareProfessionalID string `json:"healthcareProfessionalID"`
	Organization             string `json:"organization"`
	CreatedDate              int64  `json:"createdDate"`
	ExpirationDate           int64  `json:"expirationDate"`
}

// Vamos obter todos os pedidos efetuados ao paciente.
func (c *PatientContract) GetRequests(ctx contractapi.TransactionContextInterface, patientID string) ([]Request, error) {
	queryString := fmt.Sprintf(`{
        "selector": {
            "patientID": "%s"
        }
    }`, patientID)

	// Execute the query
	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}

	defer queryResultsIterator.Close()

	// Iterate over the query results and parse them into Request structs
	var requests []Request
	for queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("error retrieving next query result: %v", err)
		}

		// Unmarshal the query response into a Request struct
		var request Request
		err = json.Unmarshal(queryResponse.Value, &request)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling query result: %v", err)
		}

		// Append the request to the list of requests
		requests = append(requests, request)
	}

	return requests, nil
}

// Função que permite o paciente aceitar ou negar o pedido de acesso aos seus dados
func (c *PatientContract) AnswerRequest(ctx contractapi.TransactionContextInterface,
	response int, requestID, patientID string) error {

	// Colocar os erros de parametros 1.o não vale a pena ir à blockchain quando os campos nem válidos estão.
	if requestID == "" {
		return fmt.Errorf("invalid request ID: %s", requestID)
	}

	if patientID == "" {
		return fmt.Errorf("social security number cannot be empty")
	}

	queryString := fmt.Sprintf(`{
        "selector": {
            "patientID": "%s",
			"requestID": "%s",
        }
    }`, patientID, requestID)

	// Execute the query
	queryResultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}

	defer queryResultsIterator.Close()

	// Iterate over the query results and parse them into Request structs
	for queryResultsIterator.HasNext() {
		queryResponse, err := queryResultsIterator.Next()
		if err != nil {
			return fmt.Errorf("error retrieving next query result: %v", err)
		}

		// Unmarshal the query response into a Request struct
		var request Request
		err = json.Unmarshal(queryResponse.Value, &request)
		if err != nil {
			return fmt.Errorf("error unmarshalling query result: %v", err)
		}

		request.Status = response
		request.StatusChangedDate = time.Now().Unix()

		// Marshal the updated request back to JSON
		updatedRequestJSON, err := json.Marshal(request)
		if err != nil {
			return fmt.Errorf("failed to marshal updated request: %v", err)
		}

		// Update the request on the ledger
		err = ctx.GetStub().PutState(queryResponse.Key, updatedRequestJSON)
		if err != nil {
			return fmt.Errorf("failed to update request: %v", err)
		}
	}

	return nil
}

// Vamos obter todo o histórico do utente.
func (c *PatientContract) GetPatientMedicalHistory(ctx contractapi.TransactionContextInterface,
	patientID, healthcareProfessionalID, organization string) ([]HealthRecord, error) {

	healthRecordsKey := fmt.Sprintf("healthRecords_%s", patientID)
	existingRecordsBytes, err := ctx.GetStub().GetState(healthRecordsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing health records: %v", err)
	}

	var existingRecords []HealthRecord
	if existingRecordsBytes != nil {
		if err := json.Unmarshal(existingRecordsBytes, &existingRecords); err != nil {
			return nil, fmt.Errorf("failed to unmarshal existing health records: %v", err)
		}
	}

	return existingRecords, nil
}
