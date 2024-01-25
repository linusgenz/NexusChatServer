package helper

import ("github.com/pion/webrtc/v3"
"fmt")

func MapStringToSDPType(typeStr string) (webrtc.SDPType, error) {
	switch typeStr {
	case "offer":
		return webrtc.SDPTypeOffer, nil
	case "answer":
		return webrtc.SDPTypeAnswer, nil
	case "pranswer":
		return webrtc.SDPTypePranswer, nil
	case "rollback":
		return webrtc.SDPTypeRollback, nil
	default:
		return webrtc.SDPType(0), fmt.Errorf("invalid SDP type: %s", typeStr)
	}
}