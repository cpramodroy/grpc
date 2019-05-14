// This file registers with grpc service. This file was auto-generated by mashling at
// 2019-05-14 15:40:20.19571578 +0530 IST m=+0.046342840
package bidiproxygw

import (
	
	"log"
	"errors"
	servInfo "github.com/project-flogo/grpc/trigger"
	"google.golang.org/grpc"
)



type serviceImplpetstorebidiPetStoreServiceBidiserver struct {
	trigger *servInfo.Trigger
	serviceInfo *servInfo.ServiceInfo
}

var serviceInfopetstorebidiPetStoreServiceBidiserver = &servInfo.ServiceInfo{
	ProtoName: "petstorebidi",
	ServiceName: "PetStoreServiceBidi",
}

func init() {
	servInfo.ServiceRegistery.RegisterServerService(&serviceImplpetstorebidiPetStoreServiceBidiserver{serviceInfo: serviceInfopetstorebidiPetStoreServiceBidiserver})
}

// RunRegisterServerService registers server method implimentaion with grpc
func (s *serviceImplpetstorebidiPetStoreServiceBidiserver) RunRegisterServerService(serv *grpc.Server, trigger *servInfo.Trigger) {
	service := &serviceImplpetstorebidiPetStoreServiceBidiserver{
		trigger: trigger,
		serviceInfo: serviceInfopetstorebidiPetStoreServiceBidiserver,
	}
	RegisterPetStoreServiceBidiServer(serv, service)
}

func (s *serviceImplpetstorebidiPetStoreServiceBidiserver) BulkUsers(bdReq PetStoreServiceBidi_BulkUsersServer) error {

	methodName := "BulkUsers"

	grpcData := make(map[string]interface{})
	grpcData["methodName"] = methodName
	grpcData["strmReq"] = bdReq

	_, data, err := s.trigger.CallHandler(grpcData)

	if err != nil {
		log.Println("error: ", err)
		return err
	}

	if data != nil && data.(map[string]interface{})["error"] != nil {
		log.Println("error from end server: ", data.(map[string]interface{})["error"])
		return errors.New(data.(map[string]interface{})["error"].(string))
	}
	return nil
}

func (s *serviceImplpetstorebidiPetStoreServiceBidiserver) ServiceInfo() *servInfo.ServiceInfo {
	return s.serviceInfo
}
