package core

import (
	"edge/model"
	"edge/service"
	"fmt"
)

func newThing(guid string, vendor, m string, c model.Converter, v model.Observer) (model.Thing, error) {

	if vendor != "ztnet" {
		return nil, fmt.Errorf("unsupported vendor: %s", vendor)
	}

	var thing model.Thing

	switch m {
	case model.DeviceModelMotor:
		thing = model.NewMotor(guid, c, v)
	case model.DeviceModelDoor:
		thing = model.NewDoor(guid, c, v)
	case model.DeviceModelBody:
		thing = model.NewBodySensor(guid, c, v)
	case model.DeviceModelAM6108:
		thing = model.NewAm6108(guid, c, v)
	// case DEVICE_TYPE_BREAKER:
	// 	thing = &Breaker{Addr: 0x01}
	// case DEVICE_TYPE_BREAKER_N:
	// 	thing = &BreakerN{}
	case model.DeviceModelStb3125r:
		thing = model.NewBreaker_STB3_125_R(guid, c, v)
	case model.DeviceModelStb3125rj:
		thing = model.NewBreaker_STB3_125_RJ(guid, c, v)
	// case DEVICE_TYPE_IRACC:
	// 	thing = &Iracc{}
	// case DEVICE_TYPE_IRACC_GATEWAY:
	// 	thing = &IraccGateway{}
	case model.DeviceModelEValve:
		thing = model.NewEValve(guid, c, v)
	case model.DeviceModelLight:
		thing = model.NewLight(guid, c, v)
	case model.DeviceModelSoilSensor:
		thing = model.NewSoilSensor(guid, c, v)
	case model.DeviceModelBodyV4:
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
	case model.DeviceModelMotorFr:
		thing = model.NewMotorFR(guid, c, v)
	case model.DeviceModelRainSensor:
		thing = model.NewRainSensor(guid, c, v)
	case model.DeviceModelMotorCurtain:
		thing = model.NewMotorCurtain(guid, c, v)
	case model.DeviceModelLightModule4:
		thing = model.NewLightModule4(guid, c, v)
	case model.DeviceModelLightModule8:
		thing = model.NewLightModule8(guid, c, v)
	case model.DeviceModelLightModule16:
		thing = model.NewLightModule16(guid, c, v)
	case model.DeviceModelLoraPanel:
		thing = model.NewLoraPanel(guid, c, v)
	case model.DeviceModelR1016:
		thing = model.NewR1016(guid, c, v)
	case model.DeviceModelSerialPanel:
		thing = model.NewSerialPanel(guid, c, v)
	case model.DeviceModelElectricMeter:
		thing = model.NewElectricMeter(guid, c, v)
	case model.DeviceModelElectricMeterN:
		thing = model.NewElectricMeterN(guid, c, v)

	default:
		return nil, fmt.Errorf("new thing error")

	}

	if shadow, ok := thing.(model.Shadow); ok {

		topic := fmt.Sprintf("%v/shadow/update/delta", guid)
		service.GetMqttService().AddTopicHandler(topic, shadow.UpdateDelta)
		service.GetMqttService().AddSubscriptionTopic(topic, 1)

		topic = fmt.Sprintf("%v/shadow/get/accepted", guid)
		service.GetMqttService().AddTopicHandler(topic, shadow.GetAccepted)
		service.GetMqttService().AddSubscriptionTopic(topic, 1)

	}

	if cmd, ok := thing.(model.Command); ok {

		topic := fmt.Sprintf("commands/%v", guid)
		service.GetMqttService().AddTopicHandler(topic, cmd.CommandRequest)
		service.GetMqttService().AddSubscriptionTopic(topic, 0)

	}

	return thing, nil
}
