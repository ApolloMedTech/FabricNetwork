package main

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Colocar num ficheiro à parte, são configurações para encontrar o certificado.
// Vamos manter simples por agora, por isso vamos utilizar a rede de testes.
const (
	mspID        = "Org1MSP"
	cryptoPath   = "../../test-network/organizations/peerOrganizations/org1.example.com"
	certPath     = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/cert.pem"
	keyPath      = cryptoPath + "/users/User1@org1.example.com/msp/keystore/"
	tlsCertPath  = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint = "localhost:7051"
	gatewayPeer  = "peer0.org1.example.com"
)

var now = time.Now()

// AccessControls representa os controles de acesso do paciente
type AccessControls struct {
	AccessControls []AccessControl `json:"accessControls"`
}

// AccessControl representa um controle de acesso individual
type AccessControl struct {
	Description string `json:"description"`
	CreatedDate int64  `json:"createDate"`
	Date        int64  `json:"date"`
	EntityName  string `json:"entityName"`
	RecordType  string `json:"type"`
	RequestID   int    `json:"requestID"`
	Status      Status `json:"status"`
}

// Status representa o estado do pedido de acesso
type Status int

const (
	Pending Status = iota
	Accepted
	Denied
)

func main() {
	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "patient"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname
	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	AddPatientMedicalRecord(contract, "O Manuel partiu a unha do pé a fugir da bongo.", "Antonio",
										 "29291230", "Hospital", "Medicina interna",
										 34080)
	// Solicitar acesso aos dados do paciente
	RequestAccess(contract, "testeee", "Hospital", "29291230", "Antonio", 301290)

	// É respondido por parte do utente que o pedido pode ir lá
	AnswerRequest(contract, 1, "f7be5fefbe07d035b3f9369f6f93c364a298575017a68e027baa202077af4dc5", "29291230")

	GetRequests(contract, "29291230")
	
	GetMedicalHistory(contract, "Hospital", "Antonio", "29291230")
}

// Submit a transaction synchronously, blocking until it has been committed to the ledger.
// Relembro que estas chamadas só retornam quando a ledger é atualizada, isto é,
// A transacção completou todo o circuito.
func AddPatientMedicalRecord(contract *client.Contract, content, healthcareProfessional, socialSecurityNumber, organization, recordType string, date int64) {
	fmt.Printf("\n--> Submit Transaction: Criar uma linha na blockchain com dados médicos. \n")

	// Quando queremos submeter uma transação para o chaincode fazemos desta forma.
	// Colocar como 1º parametro o nome do método que vai ser chamado no chaincode.
	// Sempre que vamos alterar a bockchain utilizamos o método SubmitTransaction.
	dateString := int64ToString(date)

	_, err := contract.SubmitTransaction("AddPatientMedicalRecord", content, healthcareProfessional, socialSecurityNumber, organization, recordType, dateString)

	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

// Evaluate a transaction to query ledger state.
func GetMedicalHistory(contract *client.Contract, organization, healthcareProfessional, socialSecurityNumber string) {
	fmt.Println("\n--> Evaluate Transaction: Vamos obter o histórico médico mediante um NISS")

	evaluateResult, err := contract.EvaluateTransaction("GetMedicalHistory", organization, healthcareProfessional, socialSecurityNumber)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

// Evaluate a transaction to query ledger state.
func GetRequests(contract *client.Contract, socialSecurityNumber string) {
	fmt.Println("\n--> Evaluate Transaction: Vamos obter os pedidos mediante um NISS")

	evaluateResult, err := contract.EvaluateTransaction("GetRequests", socialSecurityNumber)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

// Enviar uma transação para solicitar acesso aos dados de um paciente
func RequestAccess(contract *client.Contract, description, organization, socialSecurityNumber, healthcareProfessional string, expirationDate int64) {
	fmt.Printf("\n--> Submeter Transação: Solicitar acesso aos dados de um paciente.\n")

	dateString := int64ToString(expirationDate)
	// Submeter uma transação para o chaincode
	_, err := contract.SubmitTransaction("RequestAccess", description, organization, socialSecurityNumber, healthcareProfessional, dateString)
	if err != nil {
		panic(fmt.Errorf("falha ao submeter a transação: %w", err))
	}

	fmt.Printf("*** Transação submetida com sucesso\n")
}

// Responder a um pedido de acesso aos dados do paciente
func AnswerRequest(contract *client.Contract, response int, requestID, socialSecurityNumber string) {
	fmt.Printf("\n--> Submeter Transação: Responder a um pedido de acesso aos dados do paciente.\n")

	// Converter o requestID para uma string
	//requestIDString := strconv.Itoa(requestID)

	// Converter o valor de Status para uma string
	// responseString := statusToString(response)
	
	responseString := intToString(response)

	// Submeter uma transação para o chaincode
	_, err := contract.SubmitTransaction("AnswerRequest", responseString, requestID, socialSecurityNumber)
	if err != nil {
		panic(fmt.Errorf("falha ao submeter a transação: %w", err))
	}

	fmt.Printf("*** Transação submetida com sucesso\n")
}

func intToString(i int) string {
    return strconv.Itoa(i)
}

// Format JSON data
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}

// newGrpcConnection creates a gRPC connection to the Gateway server.
func newGrpcConnection() *grpc.ClientConn {

	certificate, err := loadCertificate(tlsCertPath)

	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
func newIdentity() *identity.X509Identity {
	certificate, err := loadCertificate(certPath)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

func loadCertificate(filename string) (*x509.Certificate, error) {
	certificatePEM, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}
	return identity.CertificateFromPEM(certificatePEM)
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
func newSign() identity.Sign {
	files, err := os.ReadDir(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key directory: %w", err))
	}
	privateKeyPEM, err := os.ReadFile(path.Join(keyPath, files[0].Name()))

	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}