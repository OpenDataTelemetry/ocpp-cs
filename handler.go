package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// find the charger id by the charger number and the charger station
func defineDeviceId(chargePointId string, connectorId string) string {
	var v string
	ok := false
	if chargePointId == "EVSE_1" {
		v, ok = EVSE1[connectorId]
	} else if chargePointId == "Simulador" {
		v, ok = SimuladorCarregador[connectorId]
	} else if chargePointId == "Simulador2" {
		v, ok = SimuladorCarregador2[connectorId]
	}
	if !ok {
		// v = "DeviceId"
		v = "erro"
	}

	return v
}

func defineMQTTTopic(deviceId string) string {

	var messageTopic strings.Builder
	messageTopic.WriteString(path)
	messageTopic.WriteString(deviceId)
	messageTopic.WriteString(`/rx`)

	return messageTopic.String()
}

var (
	// <- Create var for channel
	c             chan string
	c2            chan [2]string
	sbMqttMessage strings.Builder
	// Registration of transactions
	Transaction = map[string]string{}

	// Charger stations
	EVSE1 = map[string]string{
		"0": "EVSE_1",
		"1": "19400577",
		"2": "19743013",
	}
	//Simulator
	SimuladorCarregador = map[string]string{
		"0": "Simulador",
		"1": "Simulador-1",
	}
	//Simulator
	SimuladorCarregador2 = map[string]string{
		"0": "Simulador2",
		"1": "Simulador2-1",
	}

	// path = "OpenDataTelemetry/IMT/EVSE/"
	path = "IMT/EVSE/"
	// OpenDataTelemetry/IMT/EVSE/{DeviceId}/rx

)

// HANDLER INIT
var (
	nextTransactionId = 0
)

// TransactionInfo contains info about a transaction
type TransactionInfo struct {
	id          int
	startTime   *types.DateTime
	endTime     *types.DateTime
	startMeter  int
	endMeter    int
	connectorId int
	idTag       string
}

func (ti *TransactionInfo) hasTransactionEnded() bool {
	return ti.endTime != nil && !ti.endTime.IsZero()
}

// ConnectorInfo contains status and ongoing transaction ID for a connector
type ConnectorInfo struct {
	status             core.ChargePointStatus
	currentTransaction int
}

func (ci *ConnectorInfo) hasTransactionInProgress() bool {
	return ci.currentTransaction >= 0
}

// ChargePointState contains some simple state for a connected charge point
type ChargePointState struct {
	status            core.ChargePointStatus
	diagnosticsStatus firmware.DiagnosticsStatus
	firmwareStatus    firmware.FirmwareStatus
	connectors        map[int]*ConnectorInfo // No assumptions about the # of connectors
	transactions      map[int]*TransactionInfo
	errorCode         core.ChargePointErrorCode
}

func (cps *ChargePointState) getConnector(id int) *ConnectorInfo {
	ci, ok := cps.connectors[id]
	if !ok {
		ci = &ConnectorInfo{currentTransaction: -1}
		cps.connectors[id] = ci
	}
	return ci
}

// CentralSystemHandler contains some simple state that a central system may want to keep.
// In production this will typically be replaced by database/API calls.
type CentralSystemHandler struct {
	chargePoints map[string]*ChargePointState
}

// ------------- Core profile callbacks -------------

func (handler *CentralSystemHandler) OnAuthorize(chargePointId string, request *core.AuthorizeRequest) (confirmation *core.AuthorizeConfirmation, err error) {
	logDefault(chargePointId, request.GetFeatureName()).Infof("client authorized")

	return core.NewAuthorizationConfirmation(types.NewIdTagInfo(types.AuthorizationStatusAccepted)), nil
}

func (handler *CentralSystemHandler) OnBootNotification(chargePointId string, request *core.BootNotificationRequest) (confirmation *core.BootNotificationConfirmation, err error) {
	logDefault(chargePointId, request.GetFeatureName()).Infof("boot confirmed")

	return core.NewBootNotificationConfirmation(types.NewDateTime(time.Now()), defaultHeartbeatInterval, core.RegistrationStatusAccepted), nil
}

func (handler *CentralSystemHandler) OnDataTransfer(chargePointId string, request *core.DataTransferRequest) (confirmation *core.DataTransferConfirmation, err error) {
	logDefault(chargePointId, request.GetFeatureName()).Infof("received data %d", request.Data)

	// type DataTransferRequest struct {
	// 	VendorId  string      `json:"vendorId" validate:"required,max=255"`
	// 	MessageId string      `json:"messageId,omitempty" validate:"max=50"`
	// 	Data      interface{} `json:"data,omitempty"`
	// }

	// m := fmt.Sprintf(
	// 	`{"type":"%s", "VendorId":"%s", "MessageId": "%s", "timestamp": "%s"}`,
	// 	request.GetFeatureName(),
	// 	request.VendorId,
	// 	request.MessageId,
	// 	request.Data,
	// )

	// topic := defineMQTTTopic(deviceId)

	// c2 <- [2]string{topic, m}

	return core.NewDataTransferConfirmation(core.DataTransferStatusAccepted), nil

	///BATERIA
}

func (handler *CentralSystemHandler) OnHeartbeat(chargePointId string, request *core.HeartbeatRequest) (confirmation *core.HeartbeatConfirmation, err error) {
	logDefault(chargePointId, request.GetFeatureName()).Infof("heartbeat handled")

	return core.NewHeartbeatConfirmation(types.NewDateTime(time.Now())), nil

}

func (handler *CentralSystemHandler) OnMeterValues(chargePointId string, request *core.MeterValuesRequest) (confirmation *core.MeterValuesConfirmation, err error) {

	logDefault(chargePointId, request.GetFeatureName()).Infof("received meter values for connector %v. Meter values:\n", request.ConnectorId)
	for _, mv := range request.MeterValue {
		logDefault(chargePointId, request.GetFeatureName()).Printf("%v", mv)

		// Message TODO
		// sbMqttMessage.Reset()
		// sbMqttMessage.WriteString(`{"type":"`)
		// sbMqttMessage.WriteString(request.GetFeatureName())
		// sbMqttMessage.WriteString(`", "chargePointId" : "`)
		// sbMqttMessage.WriteString(chargePointId)
		// sbMqttMessage.WriteString(`"}`)

		// m := sbMqttMessage.String()
		// fmt.Printf("\n\n### OnMeterValues: %s", m)

		deviceId := defineDeviceId(chargePointId, strconv.Itoa(request.ConnectorId))

		m := fmt.Sprintf(
			`{"type":"%s", "value":"%s", "timestamp": "%s", "unit": "%s", "format": "%s", "measurand":"%s", "context": "%s", "location": "%s", "deviceId": "%s"}`,
			request.GetFeatureName(),
			mv.SampledValue[0].Value,
			mv.Timestamp.String(),
			mv.SampledValue[0].Unit,
			mv.SampledValue[0].Format,
			mv.SampledValue[0].Measurand,
			mv.SampledValue[0].Context,
			mv.SampledValue[0].Location,
			deviceId,
		)

		topic := defineMQTTTopic(deviceId)

		c2 <- [2]string{topic, m}
	}

	return core.NewMeterValuesConfirmation(), nil
}

func (handler *CentralSystemHandler) OnStatusNotification(chargePointId string, request *core.StatusNotificationRequest) (confirmation *core.StatusNotificationConfirmation, err error) {
	info, ok := handler.chargePoints[chargePointId]
	if !ok {
		return nil, fmt.Errorf("unknown charge point %v", chargePointId)
	}
	info.errorCode = request.ErrorCode
	if request.ConnectorId > 0 {
		connectorInfo := info.getConnector(request.ConnectorId)
		connectorInfo.status = request.Status
		logDefault(chargePointId, request.GetFeatureName()).Infof("connector %v updated status to %v", request.ConnectorId, request.Status)
	} else {
		info.status = request.Status
		logDefault(chargePointId, request.GetFeatureName()).Infof("all connectors updated status to %v", request.Status)
	}

	deviceId := defineDeviceId(chargePointId, strconv.Itoa(request.ConnectorId))

	m := fmt.Sprintf(
		`{"type":"%s", "connectorId":"%s", "timestamp": "%s", "status": "%s", "errorCode": "%s", "info":"%s" , "vendorId": "%s","vendorErrorCode":"%s"}`,
		request.GetFeatureName(),
		strconv.Itoa(request.ConnectorId),
		request.Timestamp,
		request.Status,
		request.ErrorCode,
		request.Info,
		request.VendorId,
		request.VendorErrorCode,
	)

	topic := defineMQTTTopic(deviceId)

	c2 <- [2]string{topic, m}

	return core.NewStatusNotificationConfirmation(), nil
}

func (handler *CentralSystemHandler) OnStartTransaction(chargePointId string, request *core.StartTransactionRequest) (confirmation *core.StartTransactionConfirmation, err error) {
	info, ok := handler.chargePoints[chargePointId]
	if !ok {
		return nil, fmt.Errorf("unknown charge point %v", chargePointId)
	}
	connector := info.getConnector(request.ConnectorId)
	if connector.currentTransaction >= 0 {
		return nil, fmt.Errorf("connector %v is currently busy with another transaction", request.ConnectorId)
	}
	transaction := &TransactionInfo{}
	transaction.idTag = request.IdTag               //idTag: O ID da tag do usuário que iniciou a transação.
	transaction.connectorId = request.ConnectorId   //connectorId: O ID do conector onde a transação está ocorrendo.
	transaction.startMeter = request.MeterStart     //startMeter: A leitura inicial do medidor no início da transação.
	transaction.startTime = request.Timestamp       //startTime: O timestamp indicando quando a transação foi iniciada.
	transaction.id = nextTransactionId              //id: Um identificador único para a transação. Este é incrementado usando nextTransactionId.
	nextTransactionId += 1                          //
	connector.currentTransaction = transaction.id   //
	info.transactions[transaction.id] = transaction //

	// type TransactionInfo struct {
	// id          int
	// startTime   *types.DateTime
	// endTime     *types.DateTime			Ñ
	// startMeter  int						ok
	// endMeter    int						Ñ
	// connectorId int						ok
	// idTag       string					ok
	// }
	logDefault(chargePointId, request.GetFeatureName()).Infof("started transaction %v for connector %v", transaction.id, transaction.connectorId)

	deviceId := defineDeviceId(chargePointId, strconv.Itoa(request.ConnectorId))

	m := fmt.Sprintf(
		`{"type":"%s","startMeter":"%s", "transactionId": "%s", "startTime": "%s", "connectorId" : "%s", "IdTag" : "%s"}`,
		request.GetFeatureName(),
		strconv.Itoa(transaction.startMeter),
		strconv.Itoa(transaction.id),
		fmt.Sprint(transaction.startTime),
		strconv.Itoa(request.ConnectorId),
		transaction.idTag,
	)

	topic := defineMQTTTopic(deviceId)

	c2 <- [2]string{topic, m}

	//saving Connectorid from the transaction ID
	Transaction[strconv.Itoa(transaction.id)] = strconv.Itoa(request.ConnectorId)

	// Send mensage to connector
	// request := core.NewDataTransferConfirmation
	// err := centralSystem.SendRequestAsync("clientId", request, callbackFunction)
	// if err != nil {
	// log.Printf("error sending message: %v", err)
	// }

	return core.NewStartTransactionConfirmation(types.NewIdTagInfo(types.AuthorizationStatusAccepted), transaction.id), nil
}

func (handler *CentralSystemHandler) OnStopTransaction(chargePointId string, request *core.StopTransactionRequest) (confirmation *core.StopTransactionConfirmation, err error) {
	info, ok := handler.chargePoints[chargePointId]
	if !ok {
		return nil, fmt.Errorf("unknown charge point %v", chargePointId)
	}
	transaction, ok := info.transactions[request.TransactionId]
	if ok {
		connector := info.getConnector(transaction.connectorId)
		connector.currentTransaction = -1
		transaction.endTime = request.Timestamp
		transaction.endMeter = request.MeterStop
		//TODO: bill charging period to client
	}

	// type TransactionInfo struct {
	// id          int
	// startTime   *types.DateTime
	// endTime     *types.DateTime			Ñ
	// startMeter  int						ok
	// endMeter    int						Ñ
	// connectorId int						ok
	// idTag       string					ok
	// }
	logDefault(chargePointId, request.GetFeatureName()).Infof("stopped transaction %v - %v", request.TransactionId, request.Reason)
	for _, mv := range request.TransactionData {
		logDefault(chargePointId, request.GetFeatureName()).Printf("%v", mv)
	}

	deviceId := defineDeviceId(chargePointId, Transaction[strconv.Itoa(request.TransactionId)])

	m := fmt.Sprintf(
		`{"type":"%s","endMeter":"%s", "transactionId": "%s", "endTime": "%s", "connectorId" : "%s"}`,
		request.GetFeatureName(),
		strconv.Itoa(request.MeterStop),
		strconv.Itoa(request.TransactionId),
		fmt.Sprint(request.Timestamp),
		Transaction[strconv.Itoa(request.TransactionId)],
	)

	topic := defineMQTTTopic(deviceId)

	c2 <- [2]string{topic, m}

	delete(Transaction, strconv.Itoa(request.TransactionId))

	return core.NewStopTransactionConfirmation(), nil
}

// ------------- Firmware management profile callbacks -------------

func (handler *CentralSystemHandler) OnDiagnosticsStatusNotification(chargePointId string, request *firmware.DiagnosticsStatusNotificationRequest) (confirmation *firmware.DiagnosticsStatusNotificationConfirmation, err error) {
	info, ok := handler.chargePoints[chargePointId]
	if !ok {
		return nil, fmt.Errorf("unknown charge point %v", chargePointId)
	}
	info.diagnosticsStatus = request.Status
	logDefault(chargePointId, request.GetFeatureName()).Infof("updated diagnostics status to %v", request.Status)
	return firmware.NewDiagnosticsStatusNotificationConfirmation(), nil
}

func (handler *CentralSystemHandler) OnFirmwareStatusNotification(chargePointId string, request *firmware.FirmwareStatusNotificationRequest) (confirmation *firmware.FirmwareStatusNotificationConfirmation, err error) {
	info, ok := handler.chargePoints[chargePointId]
	if !ok {
		return nil, fmt.Errorf("unknown charge point %v", chargePointId)
	}
	info.firmwareStatus = request.Status
	logDefault(chargePointId, request.GetFeatureName()).Infof("updated firmware status to %v", request.Status)
	return &firmware.FirmwareStatusNotificationConfirmation{}, nil
}

// No callbacks for Local Auth management, Reservation, Remote trigger or Smart Charging profile on central system

// Utility functions

func logDefault(chargePointId string, feature string) *logrus.Entry {
	return log.WithFields(logrus.Fields{"client": chargePointId, "message": feature})
	// return log.WithFields(logrus.Fields{})
}
