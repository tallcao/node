package core

import (
	"edge/model"
	"fmt"
)

func newThing(guid string, t model.DEVICE_TYPE, c model.Converter, v model.Observer) (model.Thing, error) {

	var thing model.Thing

	switch t {
	// case model.DEVICE_TYPE_MOTOR:
	// 	thing = model.NewMotor(guid, c, v)
	case model.DEVICE_TYPE_DOOR:
		thing = model.NewDoor(guid, c, v)
	case model.DEVICE_TYPE_BODY:
		thing = model.NewBodySensor(guid, c, v)
	case model.DEVICE_TYPE_AM6108:
		thing = model.NewAm6108(guid, c, v)
	// case DEVICE_TYPE_BREAKER:
	// 	thing = &Breaker{Addr: 0x01}
	// case DEVICE_TYPE_BREAKER_N:
	// 	thing = &BreakerN{}
	case model.DEVICE_TYPE_BREAKER_STB3_125_R:
		thing = model.NewBreaker_STB3_125_R(guid, c, v)
	case model.DEVICE_TYPE_BREAKER_STB3_125_RJ:
		thing = model.NewBreaker_STB3_125_RJ(guid, c, v)
	// case DEVICE_TYPE_IRACC:
	// 	thing = &Iracc{}
	// case DEVICE_TYPE_IRACC_GATEWAY:
	// 	thing = &IraccGateway{}
	case model.DEVICE_TYPE_E_VALVE:
		thing = model.NewEValve(guid, c, v)
	case model.DEVICE_TYPE_LIGHT:
		thing = model.NewLight(guid, c, v)
	case model.DEVICE_TYPE_SOIL_SENSOR:
		thing = model.NewSoilSensor(guid, c, v)
	case model.DEVICE_TYPE_BODY_V4:
		thing = model.NewBodySensorV4(guid, c, v)
		// case DEVICE_TYPE_VOLTAGE_MODULE:
		// 	thing = &VoltageModule{}
		// case DEVICE_TYPE_METER:
		// 	thing = &Meter{Addr: 0x01}
		// case DEVICE_TYPE_RELAY_16:
		// 	thing = NewRelay16(0x01)
		// case DEVICE_TYPE_RELAY_8:
		// 	thing = NewRelay16(0x01)

		// case DEVICE_TYPE_BREAKER_STB3L_125_R:
		// 	thing = &Breaker_STB3L_125_R{Addr: 0x01}
		// case DEVICE_TYPE_BREAKER_STB3L_125_RJ:
		// 	thing = &Breaker_STB3L_125_RJ{Addr: 0x01}
	// case model.DEVICE_TYPE_MOTOR_FR:
	// 	thing = model.NewMotorFR(guid, c, v)
	case model.DEVICE_TYPE_RAIN:
		thing = model.NewRainSensor(guid, c, v)
	// case model.DEVICE_TYPE_MOTOR_CURTAIN:
	// 	thing = model.NewMotorCurtain(guid, c, v)
	case model.DEVICE_TYPE_LIGHTING_MODULE_4:
		thing = model.NewLightModule4(guid, c, v)
	case model.DEVICE_TYPE_LIGHTING_MODULE_8:
		thing = model.NewLightModule8(guid, c, v)
	case model.DEVICE_TYPE_LIGHTING_MODULE_16:
		thing = model.NewLightModule16(guid, c, v)
	case model.DEVICE_TYPE_LORA_PANEL:
		thing = model.NewLoraPanel(guid, c, v)
	default:
		return nil, fmt.Errorf("new thing error")

	}

	return thing, nil
}
