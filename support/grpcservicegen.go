package support

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/golang/protobuf/protoc-gen-go/generator"
)

const (
	serviceName = "\nservice "
)

var (
	protoPath     string
	protoFileName string
	protoImpPath  string
	appPath       string
	cmdExePath    string
)

// MethodInfoTree holds method information
type MethodInfoTree struct {
	MethodName    string
	MethodReqName string
	MethodResName string
	serviceName   string
}

// ProtoData holds proto file data
type ProtoData struct {
	Timestamp              time.Time
	Package                string
	UnaryMethodInfo        []MethodInfoTree
	ClientStreamMethodInfo []MethodInfoTree
	ServerStreamMethodInfo []MethodInfoTree
	BiDiStreamMethodInfo   []MethodInfoTree
	AllMethodInfo          []MethodInfoTree
	ProtoImpPath           string
	RegServiceName         string
	ProtoName              string
	Option                 string
	Stream                 bool
}

// AssignValues will set fullpath value
func AssignValues(path string) {
	appPath = path
}

// GenerateSupportFiles creates auto genearted code
func GenerateSupportFiles(packageName, path string) error {

	path, _ = filepath.Abs(path)
	_, err := os.Stat(path)
	if err != nil {
		log.Fatal("file path provided is invalid")
		return err
	}

	protoPath = path[:strings.LastIndex(path, string(filepath.Separator))]
	protoFileName = path[strings.LastIndex(path, string(filepath.Separator))+1:]

	log.Println("protoPath:[", protoPath, "] protoFileName:[", protoFileName, "]")

	protoImpPath = strings.Split(protoFileName, ".")[0]

	log.Println("generating pb files")
	err = generatePbFiles()
	if err != nil {
		return err
	}

	log.Println("getting proto data")
	pdArr, err := getProtoData(packageName, path)
	if err != nil {
		return err
	}

	// refactoring streaming methods and unary methods
	pdArr = arrangeProtoData(pdArr)

	log.Println("creating trigger support files")
	err = generateServiceImplFile(pdArr, "server")
	if err != nil {
		return err
	}

	log.Println("creating service support files")
	err = generateServiceImplFile(pdArr, "client")
	if err != nil {
		return err
	}

	log.Println("support files created")
	return nil
}

//server template to create trigger support files
var registryServerTemplate = template.Must(template.New("").Parse(`// This file registers with grpc service. This file was auto-generated by mashling at
// {{ .Timestamp }}
package {{.Package}}

import (
	{{if .UnaryMethodInfo}}
	"encoding/json"
	"fmt"
	"strings"
	"golang.org/x/net/context"
	{{end}}
	"log"
	"errors"
	servInfo "github.com/project-flogo/grpc/trigger"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"
)
{{$serviceName := .RegServiceName}}
{{$protoName := .ProtoName}}
{{$option := .Option}}
type serviceImpl{{$protoName}}{{$serviceName}}{{$option}} struct {
	trigger *servInfo.Trigger
	serviceInfo *servInfo.ServiceInfo
}

var serviceInfo{{$protoName}}{{$serviceName}}{{$option}} = &servInfo.ServiceInfo{
	ProtoName: "{{$protoName}}",
	ServiceName: "{{$serviceName}}",
}

func init() {
	servInfo.ServiceRegistery.RegisterServerService(&serviceImpl{{$protoName}}{{$serviceName}}{{$option}}{serviceInfo: serviceInfo{{$protoName}}{{$serviceName}}{{$option}}})
}

// RunRegisterServerService registers server method implimentaion with grpc
func (s *serviceImpl{{$protoName}}{{$serviceName}}{{$option}}) RunRegisterServerService(serv *grpc.Server, trigger *servInfo.Trigger) {
	service := &serviceImpl{{$protoName}}{{$serviceName}}{{$option}}{
		trigger: trigger,
		serviceInfo: serviceInfo{{$protoName}}{{$serviceName}}{{$option}},
	}
	Register{{$serviceName}}Server(serv, service)
}


{{- range .UnaryMethodInfo }}

func (s *serviceImpl{{$protoName}}{{$serviceName}}{{$option}}) {{.MethodName}}(ctx context.Context, req *{{.MethodReqName}}) (res *{{.MethodResName}},err error) {

	methodName := "{{.MethodName}}"
	serviceName := "{{$serviceName}}"

	grpcData := make(map[string]interface{})
	grpcData["methodName"] = methodName
	grpcData["serviceName"] = serviceName
	grpcData["contextdata"] = ctx
	grpcData["reqdata"] = req

	_, replyData, err := s.trigger.CallHandler(grpcData)

	if err != nil {
		log.Println("error: ", err)
		return nil, err
	}

	typeHandRes := fmt.Sprintf("%T", replyData)
	if strings.Compare(typeHandRes, "*status.statusError") == 0 {
		return res, replyData.(error)
	}
	typeMethodRes := fmt.Sprintf("%T", res)
	if strings.Compare(typeHandRes, typeMethodRes) == 0 {
		res = replyData.(*{{.MethodResName}})
	} else  if replyData != nil {
		var errValue = replyData.(map[string]interface{})["error"]
		if errValue != nil && len(errValue.(string)) != 0 {
			return res, errors.New(errValue.(string))
		} else {
			rDBytes, err := json.Marshal(replyData)
			if err != nil {
				log.Println("error: ", err)
				return res, err
			}
			res = &{{.MethodResName}}{}
			err = jsonpb.UnmarshalString(string(rDBytes), res)
			if err != nil {
				log.Println("error: ", err)
				return res, err
			}
		}
	} else {
		return nil, errors.New("Exception at gateway end")
	}
	//log.Println("response: ", res)

	//User implementation area

	return res, err
}

{{- end }}

{{- range .ServerStreamMethodInfo }}

func (s *serviceImpl{{$protoName}}{{$serviceName}}{{$option}}) {{.MethodName}}(req *EmptyReq, sReq {{$serviceName}}_{{.MethodName}}Server) error {

	methodName := "{{.MethodName}}"
	serviceName := "{{$serviceName}}"

	grpcData := make(map[string]interface{})
	grpcData["methodName"] = methodName
	grpcData["serviceName"] = serviceName
	grpcData["reqdata"] = req
	grpcData["strmReq"] = sReq

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

{{- end }}

{{- range .ClientStreamMethodInfo }}

func (s *serviceImpl{{$protoName}}{{$serviceName}}{{$option}}) {{.MethodName}}(cReq {{$serviceName}}_{{.MethodName}}Server) error {

	methodName := "{{.MethodName}}"
	serviceName := "{{$serviceName}}"

	grpcData := make(map[string]interface{})
	grpcData["methodName"] = methodName
	grpcData["serviceName"] = serviceName
	grpcData["strmReq"] = cReq

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

{{- end }}

{{- range .BiDiStreamMethodInfo }}

func (s *serviceImpl{{$protoName}}{{$serviceName}}{{$option}}) {{.MethodName}}(bdReq {{$serviceName}}_{{.MethodName}}Server) error {

	methodName := "{{.MethodName}}"
	serviceName := "{{$serviceName}}"

	grpcData := make(map[string]interface{})
	grpcData["methodName"] = methodName
	grpcData["serviceName"] = serviceName
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

{{- end }}

func (s *serviceImpl{{$protoName}}{{$serviceName}}{{$option}}) ServiceInfo() *servInfo.ServiceInfo {
	return s.serviceInfo
}

`))

//client template to create grpc service support file
var registryClientTemplate = template.Must(template.New("").Parse(`// This file registers with grpc service. This file was auto-generated by mashling at
	// {{ .Timestamp }}
	package {{.Package}}

	import (
		"context"
		{{if .UnaryMethodInfo}}
		"encoding/json"
		"github.com/project-flogo/grpc/support"
		{{end}}
		"errors"
		{{if .Stream}}
		"strings"
		"io"
		{{end}}
		"log"
		{{if .ServerStreamMethodInfo}}
		"github.com/imdario/mergo"
		{{end}}

		servInfo "github.com/project-flogo/grpc/activity"
		"google.golang.org/grpc"
	)
	{{$serviceName := .RegServiceName}}
	{{$protoName := .ProtoName}}
	{{$option := .Option}}
	type clientService{{$protoName}}{{$serviceName}}{{$option}} struct {
		serviceInfo *servInfo.ServiceInfo
	}

	var serviceInfo{{$protoName}}{{$serviceName}}{{$option}} = &servInfo.ServiceInfo{
		ProtoName: "{{$protoName}}",
		ServiceName: "{{$serviceName}}",
	}

	func init() {
		servInfo.ClientServiceRegistery.RegisterClientService(&clientService{{$protoName}}{{$serviceName}}{{$option}}{serviceInfo: serviceInfo{{$protoName}}{{$serviceName}}{{$option}}})
	}

	//GetRegisteredClientService returns client implimentaion stub with grpc connection
	func (cs *clientService{{$protoName}}{{$serviceName}}{{$option}}) GetRegisteredClientService(gCC *grpc.ClientConn) interface{} {
		return New{{$serviceName}}Client(gCC)
	}

	func (cs *clientService{{$protoName}}{{$serviceName}}{{$option}}) ServiceInfo() *servInfo.ServiceInfo {
		return cs.serviceInfo
	}

	func (cs *clientService{{$protoName}}{{$serviceName}}{{$option}}) InvokeMethod(reqArr map[string]interface{}) map[string]interface{} {

		clientObject := reqArr["ClientObject"].({{$serviceName}}Client)
		methodName := reqArr["MethodName"].(string)

		switch methodName {
		{{- range .AllMethodInfo }}
		case "{{.MethodName}}":
			return {{.MethodName}}(clientObject, reqArr)
		{{- end }}
		}

		resMap := make(map[string]interface{},2)
		resMap["Response"] = []byte("null")
		resMap["Error"] = errors.New("Method not Available: " + methodName)
		return resMap
	}

	{{- range .UnaryMethodInfo }}
	func {{.MethodName}}(client {{$serviceName}}Client, values interface{}) map[string]interface{} {
		req := &{{.MethodReqName}}{}
		support.AssignStructValues(req, values)
		res, err := client.{{.MethodName}}(context.Background(), req)
		b, errMarshl := json.Marshal(res)
		if errMarshl != nil {
			log.Println("Error: ", errMarshl)
			return nil
		}

		resMap := make(map[string]interface{}, 2)
		resMap["Response"] = b
		resMap["Error"] = err
		return resMap
	}
	{{- end }}

	{{- range .ServerStreamMethodInfo }}

	func {{.MethodName}}(client {{$serviceName}}Client, reqArr map[string]interface{}) map[string]interface{} {
		resMap := make(map[string]interface{}, 1)

		if reqArr["Mode"] != nil {
			mode := reqArr["Mode"].(string)
			if strings.Compare(mode,"rest-to-grpc") == 0 {
				resMap["Error"] = errors.New("streaming operation is not allowed in rest to grpc case")
				return resMap
			}
		}

		req := &{{.MethodReqName}}{}
		reqData := reqArr["reqdata"].(*{{.MethodReqName}})
		if err := mergo.Merge(req, reqData, mergo.WithOverride); err != nil {
			resMap["Error"] = errors.New("unable to merge reqData values")
			return resMap
		}

		sReq := reqArr["strmReq"].({{$serviceName}}_{{.MethodName}}Server)

		stream, err := client.{{.MethodName}}(context.Background(), req)
		if err != nil {
			log.Println("erorr while getting stream object for {{.MethodName}}:", err)
			resMap["Error"] = err
			return resMap
		}
		for {
			obj, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Println("erorr occured in {{.MethodName}} Recv():", err)
				resMap["Error"] = err
				return resMap
			}
			err = sReq.Send(obj)
			if err != nil {
				log.Println("error occured in {{.MethodName}} Send():", err)
				resMap["Error"] = err
				return resMap
			}
		}
		resMap["Error"] = nil
		return resMap
	}
	{{- end }}

	{{- range .ClientStreamMethodInfo }}

	func {{.MethodName}}(client {{$serviceName}}Client, reqArr map[string]interface{}) map[string]interface{} {
		resMap := make(map[string]interface{}, 1)

		if reqArr["Mode"] != nil {
			mode := reqArr["Mode"].(string)
			if strings.Compare(mode,"rest-to-grpc") == 0 {
				resMap["Error"] = errors.New("streaming operation is not allowed in rest to grpc case")
				return resMap
			}
		}

		stream, err := client.{{.MethodName}}(context.Background())
		if err != nil {
			log.Println("erorr while getting stream object for {{.MethodName}}:", err)
			resMap["Error"] = err
			return resMap
		}

		cReq := reqArr["strmReq"].({{$serviceName}}_{{.MethodName}}Server)

		for {
			dataObj, err := cReq.Recv()
			if err == io.EOF {
				obj, err := stream.CloseAndRecv()
				if err != nil {
					log.Println("erorr occured in {{.MethodName}} CloseAndRecv():", err)
					resMap["Error"] = err
					return resMap
				}
				resMap["Error"] = cReq.SendAndClose(obj)
				return resMap
			}
			if err != nil {
				log.Println("error occured in {{.MethodName}} Recv():", err)
				resMap["Error"] = err
				return resMap
			}

			if err := stream.Send(dataObj); err != nil {
				log.Println("error while sending dataObj with client stream:", err)
				resMap["Error"] = err
				return resMap
			}
		}
	}

	{{- end }}

	{{- range .BiDiStreamMethodInfo }}

	func {{.MethodName}}(client {{$serviceName}}Client, reqArr map[string]interface{}) map[string]interface{} {
		resMap := make(map[string]interface{}, 1)

		if reqArr["Mode"] != nil {
			mode := reqArr["Mode"].(string)
			if strings.Compare(mode,"rest-to-grpc") == 0 {
				resMap["Error"] = errors.New("streaming operation is not allowed in rest to grpc case")
				return resMap
			}
		}

		bReq := reqArr["strmReq"].({{$serviceName}}_{{.MethodName}}Server)

		stream, err := client.{{.MethodName}}(context.Background())
		if err != nil {
			log.Println("error while getting stream object for {{.MethodName}}:", err)
			resMap["Error"] = err
			return resMap
		}

		waits := make(chan struct{})
		go func() {
			for {
				obj, err := bReq.Recv()
				if err == io.EOF {
					resMap["Error"] = nil
					stream.CloseSend()
					close(waits)
					return
				}
				if err != nil {
					log.Println("error occured in {{.MethodName}} bidi Recv():", err)
					resMap["Error"] = err
					close(waits)
					return
				}
				if err := stream.Send(obj); err != nil {
					log.Println("error while sending obj with stream:", err)
					resMap["Error"] = err
					close(waits)
					return
				}
			}
		}()

		waitc := make(chan struct{})
		go func() {
			for {
				obj, err := stream.Recv()
				if err == io.EOF {
					resMap["Error"] = nil
					close(waitc)
					return
				}
				if err != nil {
					log.Println("erorr occured in {{.MethodName}} stream Recv():", err)
					resMap["Error"] = err
					close(waitc)
					return
				}
				if sdErr := bReq.Send(obj); sdErr != nil {
					log.Println("error while sending obj with bidi Send():", sdErr)
					resMap["Error"] = sdErr
					close(waitc)
					return
				}
			}
		}()
		<-waitc
		<-waits
		return resMap
	}

	{{- end }}

	`))

// Exec executes a command within the build context.
func Exec(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	if len(cmdExePath) != 0 {
		cmd.Dir = cmdExePath
		cmdExePath = ""
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal("error: ", string(output), err)
		return err
	}

	return nil
}

// generatePbFiles generates stub file based on given proto
func generatePbFiles() error {
	fullPath := filepath.Join(appPath)

	_, err := os.Stat(fullPath)
	if err != nil {
		os.MkdirAll(fullPath, os.ModePerm)
	}

	_, err = exec.LookPath("protoc")
	if err != nil {
		log.Fatal("protoc is not available")
		return err
	}

	err = Exec("protoc", "-I", protoPath, protoPath+string(filepath.Separator)+protoFileName, "--go_out=plugins=grpc:"+fullPath)
	if err != nil {
		_, statErr := os.Stat(fullPath)
		if statErr == nil {
			os.RemoveAll(fullPath)
		}
		log.Fatal("error occured", err)
		return err
	}
	return nil
}

// arrangeProtoData refactors different types of methods from all method info list
func arrangeProtoData(pdArr []ProtoData) []ProtoData {

	for index, protoData := range pdArr {
		for _, mthdInfo := range protoData.AllMethodInfo {
			clientStrm := false
			servrStrm := false

			if strings.Contains(mthdInfo.MethodReqName, "stream ") {
				mthdInfo.MethodReqName = strings.Replace(mthdInfo.MethodReqName, "stream ", "", -1)
				clientStrm = true
				protoData.Stream = true
			}
			if strings.Contains(mthdInfo.MethodResName, "stream ") {
				mthdInfo.MethodResName = strings.Replace(mthdInfo.MethodResName, "stream ", "", -1)
				servrStrm = true
				protoData.Stream = true
			}
			if !clientStrm && !servrStrm {
				protoData.UnaryMethodInfo = append(protoData.UnaryMethodInfo, mthdInfo)
			} else if clientStrm && servrStrm {
				protoData.BiDiStreamMethodInfo = append(protoData.BiDiStreamMethodInfo, mthdInfo)
			} else if clientStrm {
				protoData.ClientStreamMethodInfo = append(protoData.ClientStreamMethodInfo, mthdInfo)
			} else if servrStrm {
				protoData.ServerStreamMethodInfo = append(protoData.ServerStreamMethodInfo, mthdInfo)
			}
		}
		pdArr[index] = protoData
	}

	return pdArr
}

// getProtoData reads proto and returns proto data present in proto file
func getProtoData(packageName, protoPath string) ([]ProtoData, error) {
	var regServiceName string
	var methodInfoList []MethodInfoTree

	bytes, err := ioutil.ReadFile(protoPath)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	fullString := string(bytes)

	var ProtodataArr []ProtoData

	tempString := fullString
	for i := 0; i < strings.Count(fullString, serviceName); i++ {

		//getting service declaration full string
		tempString = tempString[strings.Index(tempString, serviceName):]

		//getting entire service declaration
		temp := tempString[:strings.Index(tempString, "}")+1]

		regServiceName = strings.TrimSpace(temp[strings.Index(temp, serviceName)+len(serviceName) : strings.Index(temp, "{")])
		regServiceName = generator.CamelCase(regServiceName)
		temp = temp[strings.Index(temp, "rpc")+len("rpc"):]

		//entire rpc methods content
		methodArr := strings.Split(temp, "rpc")

		for _, mthd := range methodArr {
			methodInfo := MethodInfoTree{}
			mthdDtls := strings.Split(mthd, "(")
			methodInfo.MethodName = generator.CamelCase(strings.TrimSpace(mthdDtls[0]))
			methodInfo.MethodReqName = generator.CamelCase(strings.TrimSpace(strings.Split(mthdDtls[1], ")")[0]))
			methodInfo.MethodResName = generator.CamelCase(strings.TrimSpace(strings.Split(mthdDtls[2], ")")[0]))
			methodInfo.serviceName = regServiceName
			methodInfoList = append(methodInfoList, methodInfo)
		}
		protodata := ProtoData{
			Package:        packageName,
			AllMethodInfo:  methodInfoList,
			Timestamp:      time.Now(),
			ProtoImpPath:   protoImpPath,
			RegServiceName: regServiceName,
			ProtoName:      strings.Split(protoFileName, ".")[0],
		}

		ProtodataArr = append(ProtodataArr, protodata)
		methodInfoList = nil

		//getting next service content
		tempString = tempString[strings.Index(tempString, serviceName)+len(serviceName):]
	}

	return ProtodataArr, nil
}

// generateServiceImplFile creates implementation files supported for grpc trigger and grpc service
func generateServiceImplFile(pdArr []ProtoData, option string) error {
	dirPath := filepath.Join(appPath)
	_, fileErr := os.Stat(dirPath)
	if fileErr != nil {
		os.MkdirAll(dirPath, os.ModePerm)
	}
	for _, pd := range pdArr {
		connectorFile := filepath.Join(appPath, strings.Split(protoFileName, ".")[0]+"."+pd.RegServiceName+"."+option+".grpcservice.go")
		f, err := os.Create(connectorFile)
		if err != nil {
			log.Fatal("error: ", err)
			return err
		}
		defer f.Close()
		pd.Option = option
		if strings.Compare(option, "server") == 0 {
			err = registryServerTemplate.Execute(f, pd)
		} else {
			err = registryClientTemplate.Execute(f, pd)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
