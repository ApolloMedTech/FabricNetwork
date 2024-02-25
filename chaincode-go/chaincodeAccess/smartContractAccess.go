package chaincodeAccess

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type PatientAccessControl struct {
	contractapi.Contract
}

type AccessControls struct {
	AccessControls []AccessControl `json:"accessControls"`
}

type AccessControl struct {
	Description string `json:"description"`
	CreatedDate int64  `json:"createDate"`
	Date        int64  `json:"date"`
	EntityName  string `json:"entityName"`
	RecordType  string `json:"type"`
	RequestID   int    `json:"requestID"`
	Status      Status `json:"status"`
}

// HealthcareContract representa o smartcontract
type HealthcareContract struct {
	Requests []Request `json:"requests"` // Lista de ID's dos pedidos efetuados

}

// Estrutura de Request (não sei como deveria estar a organização)
type Request struct {
	Organization string
	Patient      PatientAccessControl
	Status       Status
}

// Estrutura que define o estado do pedido, começa por estar pendente, depois é aceite o negado.
type Status int

const (
	Pending Status = iota
	Accepted
	Denied
)

// Enviar um pedido ao cliente
func (hc *HealthcareContract) SendRequest(organization string, patient PatientAccessControl, socialSecurityNumber string, ctx contractapi.TransactionContextInterface, pac *PatientAccessControl) {
	// Cria uma instancia de Request e adiciona à lista de de ID's de pedidos efetuados
	request := Request{
		Organization: organization,
		Patient:      patient,
		Status:       Pending,
	}

	hc.Requests = append(hc.Requests, request)

	// Adicionar acessos à carteira do paciente quando o pedido é enviado
	err := pac.addAccessControl(ctx, "Content", socialSecurityNumber, "EntityName", "RecordType", time.Now().Unix())
	if err != nil {
		fmt.Println("Impossible to send an access request to:", request.Patient)
	}
}

// Função que permite o paciente aceitar ou negar o pedido de acesso aos seus dados
func (hc *HealthcareContract) RespondToRequest(ctx contractapi.TransactionContextInterface, requestID int, response Status, socialSecurityNumber string) error {
	// Encontrar o pedido com o ID correspondente
	var request *Request
	for i := range hc.Requests {
		if i == requestID {
			request = &hc.Requests[i]
			break
		}
	}

	if request == nil {
		return fmt.Errorf("Request not found")
	}

	// Iterar sobre os acessos associados a este pedido
	accessControls, err := getAccessControls(ctx, socialSecurityNumber)
	if err != nil {
		return fmt.Errorf("failed to retrieve patient's access controls: %v", err)
	}

	// Atualizar o estado dos acessos associados ao pedido
	for i, accessControl := range accessControls.AccessControls {
		if accessControl.RequestID == requestID {
			accessControl.Status = response
			accessControls.AccessControls[i] = accessControl
			break
		}
	}

	// Atualizar os acessos
	if err := updateAccessControl(ctx, *accessControls, socialSecurityNumber); err != nil {
		return fmt.Errorf("failed to update patient's access controls: %v", err)
	}

	if response == Denied {
		// Se o pedido for negado
		fmt.Println("Request denied for organization:", request.Organization)

	} else if response == Accepted {
		// Se o pedido for aceito
		fmt.Println("Request accepted for organization:", request.Organization)
	}

	// Volta a dar update ao estado do pedido de acesso
	request.Status = response

	return nil
}

func (c *PatientAccessControl) addAccessControl(ctx contractapi.TransactionContextInterface,
	content, socialSecurityNumber, entityName, recordType string, date int64) error {

	compositeKey, err := createCompositeKey(ctx, socialSecurityNumber)
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	accessControls, err := getAccessControls(ctx, compositeKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal patient wallet: %v", err)
	}

	//criar novo registo na lista de acessos
	newRecord := AccessControl{
		Description: content,
		CreatedDate: time.Now().Unix(),
		Date:        date,
		EntityName:  entityName,
		RecordType:  recordType,
	}

	//adicionar à lista
	accessControls.AccessControls = append(accessControls.AccessControls, newRecord)

	//atualizar a lista
	if err := updateAccessControl(ctx, *accessControls, compositeKey); err != nil {
		return fmt.Errorf("failed to update the wallet: %v", err)
	}

	return nil
}

func createCompositeKey(ctx contractapi.TransactionContextInterface, key string) (string, error) {
	compositeKey, err := ctx.GetStub().CreateCompositeKey("HealthRecord", []string{"socialSecurityNumber", key})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key: %v", err)
	}
	return compositeKey, nil
}

func getAccessControls(ctx contractapi.TransactionContextInterface, key string) (*AccessControls, error) {

	walletBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read patient wallet: %v", err)
	}

	var accessControls AccessControls

	if walletBytes != nil {
		err = json.Unmarshal(walletBytes, &accessControls)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal patient wallet: %v", err)
		}
	} else {
		// Se não houver controlos de acesso armazenados é inicializada uma nova estrutura AccessControls
		accessControls = AccessControls{AccessControls: []AccessControl{}}
	}

	return &accessControls, nil
}

// update de uma entrada da lista de acessos
// Atualiza a lista de acessos associada à chave especifíca na blockchain
func updateAccessControl(ctx contractapi.TransactionContextInterface, accessControls AccessControls, key string) error {
	accessControlsBytes, err := json.Marshal(accessControls)
	if err != nil {
		return fmt.Errorf("failed to marshal access controls: %v", err)
	}

	err = ctx.GetStub().PutState(key, accessControlsBytes)
	if err != nil {
		return fmt.Errorf("failed to update access controls: %v", err)
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
