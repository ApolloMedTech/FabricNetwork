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

	//createAsset(contract, "O Manuel partiu a unha do pé a fugir da bongo.", "29291230", "lol", "lol", 1000)
	GetMedicalHistory(contract, "29291230")
	//GetAccessControl(contract, "29291230")

	// Solicitar acesso aos dados do paciente
	SendRequest(contract, "Hospital", "29291230")

	// Respondendo a um pedido de acesso (por exemplo, aceitando o acesso)
	// Suponha que haja um pedido de acesso pendente, então vamos responder a ele
	RespondToAccessRequest(contract, 1, Accepted)
}

// Submit a transaction synchronously, blocking until it has been committed to the ledger.
// Relembro que estas chamadas só retornam quando a ledger é atualizada, isto é,
// A transacção completou todo o circuito.
func createAsset(contract *client.Contract, content, socialSecurityNumber, entityName, recordType string, date int64) {
	fmt.Printf("\n--> Submit Transaction: Criar uma linha na blockchain com dados médicos. \n")

	// Quando queremos submeter uma transação para o chaincode fazemos desta forma.
	// Colocar como 1º parametro o nome do método que vai ser chamado no chaincode.
	// Sempre que vamos alterar a bockchain utilizamos o método SubmitTransaction.
	dateString := int64ToString(date)

	_, err := contract.SubmitTransaction("AddDataToWallet", content, socialSecurityNumber, entityName, recordType, dateString)

	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

// Evaluate a transaction to query ledger state.
func GetMedicalHistory(contract *client.Contract, socialSecurityNumber string) {
	fmt.Println("\n--> Evaluate Transaction: Vamos obter o histórico médico mediante um NISS")

	evaluateResult, err := contract.EvaluateTransaction("GetMedicalHistory", socialSecurityNumber)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

// Evaluate a transaction to query ledger state.
// Evaluate a transaction to query ledger state for patient access controls.
func GetAccessControl(contract *client.Contract, socialSecurityNumber string) {
	fmt.Println("\n--> Avaliar Transação: Obtendo controles de acesso para um paciente")

	// Avaliar a transação para obter os controles de acesso para um paciente
	evaluateResult, err := contract.EvaluateTransaction("GetAccessControl", socialSecurityNumber)
	if err != nil {
		panic(fmt.Errorf("falha ao obter os controles de acesso do paciente: %v", err))
	}

	// Deserializar a resposta JSON em AccessControls
	var accessControls AccessControls
	if err := json.Unmarshal(evaluateResult, &accessControls); err != nil {
		panic(fmt.Errorf("falha ao fazer unmarshal dos controles de acesso: %v", err))
	}

	// Exibir os controles de acesso
	fmt.Println("*** Controles de Acesso:")
	for _, accessControl := range accessControls.AccessControls {
		fmt.Printf("ID do Pedido: %d, Status: %s\n", accessControl.RequestID, accessControl.Status)
	}
}

// Enviar uma transação para solicitar acesso aos dados de um paciente
func SendRequest(contract *client.Contract, organization, socialSecurityNumber string) {
	fmt.Printf("\n--> Submeter Transação: Solicitar acesso aos dados de um paciente.\n")

	// Submeter uma transação para o chaincode
	_, err := contract.SubmitTransaction("SendRequest", organization, socialSecurityNumber)
	if err != nil {
		panic(fmt.Errorf("falha ao submeter a transação: %w", err))
	}

	fmt.Printf("*** Transação submetida com sucesso\n")
}

// Responder a um pedido de acesso aos dados do paciente
func RespondToRequest(contract *client.Contract, requestID int, response Status) {
	fmt.Printf("\n--> Submeter Transação: Responder a um pedido de acesso aos dados do paciente.\n")

	// Converter o requestID para uma string
	requestIDString := strconv.Itoa(requestID)

	// Converter o valor de Status para uma string
	responseString := statusToString(response)

	// Submeter uma transação para o chaincode
	_, err := contract.SubmitTransaction("RespondToRequest", requestIDString, responseString)
	if err != nil {
		panic(fmt.Errorf("falha ao submeter a transação: %w", err))
	}

	fmt.Printf("*** Transação submetida com sucesso\n")
}

// Converte um valor de Status para uma string correspondente
func statusToString(status Status) string {
	switch status {
	case Pending:
		return "Pending"
	case Accepted:
		return "Accepted"
	case Denied:
		return "Denied"
	default:
		return ""
	}
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
