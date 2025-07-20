package mysensors

import (
	"fmt"
	"strconv"
	"strings"
)

type MessageType int

const (
	PRESENTATION MessageType = 0
	SET          MessageType = 1
	REQ          MessageType = 2
	INTERNAL     MessageType = 3
	STREAM       MessageType = 4
)

type InternalType int

const (
	I_BATTERY_LEVEL        InternalType = 0
	I_TIME                 InternalType = 1
	I_VERSION              InternalType = 2
	I_ID_REQUEST           InternalType = 3
	I_ID_RESPONSE          InternalType = 4
	I_INCLUSION_MODE       InternalType = 5
	I_CONFIG               InternalType = 6
	I_FIND_PARENT          InternalType = 7
	I_FIND_PARENT_RESPONSE InternalType = 8
	I_LOG_MESSAGE          InternalType = 9
	I_CHILDREN             InternalType = 10
	I_SKETCH_NAME          InternalType = 11
	I_SKETCH_VERSION       InternalType = 12
	I_REBOOT               InternalType = 13
	I_GATEWAY_READY        InternalType = 14
)

type SensorType int

const (
	S_DOOR                  SensorType = 0
	S_MOTION                SensorType = 1
	S_SMOKE                 SensorType = 2
	S_BINARY                SensorType = 3
	S_DIMMER                SensorType = 4
	S_COVER                 SensorType = 5
	S_TEMP                  SensorType = 6
	S_HUM                   SensorType = 7
	S_BARO                  SensorType = 8
	S_WIND                  SensorType = 9
	S_RAIN                  SensorType = 10
	S_UV                    SensorType = 11
	S_WEIGHT                SensorType = 12
	S_POWER                 SensorType = 13
	S_HEATER                SensorType = 14
	S_DISTANCE              SensorType = 15
	S_LIGHT_LEVEL           SensorType = 16
	S_ARDUINO_NODE          SensorType = 17
	S_ARDUINO_REPEATER_NODE SensorType = 18
	S_LOCK                  SensorType = 19
	S_IR                    SensorType = 20
	S_WATER                 SensorType = 21
	S_AIR_QUALITY           SensorType = 22
	S_CUSTOM                SensorType = 23
	S_DUST                  SensorType = 24
	S_SCENE_CONTROLLER      SensorType = 25
	S_RGB_LIGHT             SensorType = 26
	S_RGBW_LIGHT            SensorType = 27
	S_COLOR_SENSOR          SensorType = 28
	S_HVAC                  SensorType = 29
	S_MULTIMETER            SensorType = 30
	S_SPRINKLER             SensorType = 31
	S_WATER_LEAK            SensorType = 32
	S_SOUND                 SensorType = 33
	S_VIBRATION             SensorType = 34
	S_MOISTURE              SensorType = 35
	S_INFO                  SensorType = 36
	S_GAS                   SensorType = 37
	S_GPS                   SensorType = 38
	S_WATER_QUALITY         SensorType = 39
)

type VariableType int

const (
	V_TEMP               VariableType = 0
	V_HUM                VariableType = 1
	V_STATUS             VariableType = 2
	V_PERCENTAGE         VariableType = 3
	V_PRESSURE           VariableType = 4
	V_FORECAST           VariableType = 5
	V_RAIN               VariableType = 6
	V_RAINRATE           VariableType = 7
	V_WIND               VariableType = 8
	V_GUST               VariableType = 9
	V_DIRECTION          VariableType = 10
	V_UV                 VariableType = 11
	V_WEIGHT             VariableType = 12
	V_DISTANCE           VariableType = 13
	V_IMPEDANCE          VariableType = 14
	V_ARMED              VariableType = 15
	V_TRIPPED            VariableType = 16
	V_WATT               VariableType = 17
	V_KWH                VariableType = 18
	V_SCENE_ON           VariableType = 19
	V_SCENE_OFF          VariableType = 20
	V_HVAC_FLOW_STATE    VariableType = 21
	V_HVAC_SPEED         VariableType = 22
	V_LIGHT_LEVEL        VariableType = 23
	V_VAR1               VariableType = 24
	V_VAR2               VariableType = 25
	V_VAR3               VariableType = 26
	V_VAR4               VariableType = 27
	V_VAR5               VariableType = 28
	V_UP                 VariableType = 29
	V_DOWN               VariableType = 30
	V_STOP               VariableType = 31
	V_IR_SEND            VariableType = 32
	V_IR_RECEIVE         VariableType = 33
	V_FLOW               VariableType = 34
	V_VOLUME             VariableType = 35
	V_LOCK_STATUS        VariableType = 36
	V_LEVEL              VariableType = 37
	V_VOLTAGE            VariableType = 38
	V_CURRENT            VariableType = 39
	V_RGB                VariableType = 40
	V_RGBW               VariableType = 41
	V_ID                 VariableType = 42
	V_UNIT_PREFIX        VariableType = 43
	V_HVAC_SETPOINT_COOL VariableType = 44
	V_HVAC_SETPOINT_HEAT VariableType = 45
	V_HVAC_FLOW_MODE     VariableType = 46
)

type Message struct {
	NodeID      int
	ChildID     int
	MessageType MessageType
	Ack         bool
	SubType     int
	Payload     string
}

func ParseMessage(data string) (*Message, error) {
	parts := strings.Split(strings.TrimSpace(data), ";")
	if len(parts) != 6 {
		return nil, fmt.Errorf("invalid message format: expected 6 parts, got %d", len(parts))
	}

	nodeID, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid node ID: %w", err)
	}

	childID, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid child ID: %w", err)
	}

	msgType, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid message type: %w", err)
	}

	ack := parts[3] == "1"

	subType, err := strconv.Atoi(parts[4])
	if err != nil {
		return nil, fmt.Errorf("invalid sub type: %w", err)
	}

	return &Message{
		NodeID:      nodeID,
		ChildID:     childID,
		MessageType: MessageType(msgType),
		Ack:         ack,
		SubType:     subType,
		Payload:     parts[5],
	}, nil
}

func (m *Message) String() string {
	ack := "0"
	if m.Ack {
		ack = "1"
	}
	return fmt.Sprintf("%d;%d;%d;%s;%d;%s", m.NodeID, m.ChildID, m.MessageType, ack, m.SubType, m.Payload)
}

func (m *Message) IsInternal() bool {
	return m.MessageType == INTERNAL
}

func (m *Message) IsSet() bool {
	return m.MessageType == SET
}

func (m *Message) IsReq() bool {
	return m.MessageType == REQ
}

func (m *Message) IsPresentation() bool {
	return m.MessageType == PRESENTATION
}

func (m *Message) GetInternalType() InternalType {
	return InternalType(m.SubType)
}

func (m *Message) GetVariableType() VariableType {
	return VariableType(m.SubType)
}

func (m *Message) GetSensorType() SensorType {
	return SensorType(m.SubType)
}

func NewSetMessage(nodeID, childID int, varType VariableType, payload string) *Message {
	return &Message{
		NodeID:      nodeID,
		ChildID:     childID,
		MessageType: SET,
		Ack:         false,
		SubType:     int(varType),
		Payload:     payload,
	}
}

func NewReqMessage(nodeID, childID int, varType VariableType) *Message {
	return &Message{
		NodeID:      nodeID,
		ChildID:     childID,
		MessageType: REQ,
		Ack:         false,
		SubType:     int(varType),
		Payload:     "",
	}
}

func NewInternalMessage(nodeID int, intType InternalType, payload string) *Message {
	return &Message{
		NodeID:      nodeID,
		ChildID:     255,
		MessageType: INTERNAL,
		Ack:         false,
		SubType:     int(intType),
		Payload:     payload,
	}
}
