package chaincode

// patient contract.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type HealthContract struct {
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

// Vamos obter todo o histórico do utente.
func (c *HealthContract) GetPatientMedicalHistory(ctx contractapi.TransactionContextInterface,
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

// Vamos obter todos os pedidos efetuados ao paciente.
func (c *HealthContract) GetRequests(ctx contractapi.TransactionContextInterface, patientID string) ([]Request, error) {
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
func (c *HealthContract) AnswerRequest(ctx contractapi.TransactionContextInterface,
	response int, requestID, patientID string) error {

	// Colocar os erros de parametros 1.o não vale a pena ir à blockchain quando os campos nem válidos estão.
	if requestID == "" {
		return fmt.Errorf("invalid request ID: %s", requestID)
	}

	if patientID == "" {
		return fmt.Errorf("social security number cannot be empty")
	}

	compositeKey, err := createCompositeKey(ctx, patientID)
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
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
			fmt.Errorf("error retrieving next query result: %v", err)
		}

		// Unmarshal the query response into a Request struct
		var request Request
		err = json.Unmarshal(queryResponse.Value, &request)
		if err != nil {
			return fmt.Errorf("error unmarshalling query result: %v", err)
		}

		request.Status = response
		request.StatusChangedDate = time.Now().Unix()
	}

	// Vamos rezar para que ele seja inteligente e que esteja ligado no que diz respeito
	// a ser o mesmo objeto e desta forma poupo tempo na iteração etc.
	if err := updateWallet(ctx, *patientWallet, compositeKey); err != nil {
		return fmt.Errorf("failed to update the wallet: %v", err)
	}

	return nil
}

func findPendingRequest(requests []Request, requestID string) *Request {
	var targetRequest *Request

	for i := range requests {
		_req := requests[i]

		if _req.RequestID == requestID {
			targetRequest = &_req
			break
		}
	}

	return targetRequest
}

func createCompositeKey(ctx contractapi.TransactionContextInterface, key string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("PatientWallet", []string{"patientID", key})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key: %v", err)
	}
	return compositeKey, nil
}

// func updateWallet(ctx contractapi.TransactionContextInterface, patientWallet PatientWallet, key string) error {
// 	updatedWalletBytes, err := json.Marshal(patientWallet)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal updated patient wallet: %v", err)
// 	}

// 	err = ctx.GetStub().PutState(key, updatedWalletBytes)
// 	if err != nil {
// 		return fmt.Errorf("failed to update patient wallet: %v", err)
// 	}

// 	return nil
// }

func generateUniqueID(patientID, description string) string {
	input := patientID + description

	hasher := sha256.New()
	hasher.Write([]byte(input))
	hashInBytes := hasher.Sum(nil)

	hashString := hex.EncodeToString(hashInBytes)

	return hashString
}
