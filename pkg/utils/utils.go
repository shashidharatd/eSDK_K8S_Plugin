package utils

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/golang/glog"
)

var DoradoCloneVersion = "V600R003C00"

func PathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func ExecShellCmd(format string, args ...interface{}) (string, error) {
	cmd := fmt.Sprintf(format, args...)
	log.Infof("Gonna run shell cmd \"%s\".", cmd)

	shCmd := exec.Command("/bin/sh", "-c", cmd)
	output, err := shCmd.CombinedOutput()
	if err != nil {
		log.Warningf("Run shell cmd \"%s\" error: %s.", cmd, output)
		return string(output), err
	}

	log.Infof("Shell cmd \"%s\" result:\n%s", cmd, output)
	return string(output), nil
}

func GetLunName(name string) string {
	if len(name) <= 22 {
		return name
	}

	return name[:22]
}

func GetFusionStorageLunName(name string) string {
	if len(name) <= 95 {
		return name
	}
	return name[:95]
}

func GetFileSystemName(name string) string {
	return strings.Replace(name, "-", "_", -1)
}

func GetSharePath(name string) string {
	return "/" + strings.Replace(name, "-", "_", -1) + "/"
}

func GetHostName() (string, error) {
	hostname, err := ExecShellCmd("hostname | xargs echo -n")
	if err != nil {
		return "", err
	}

	return hostname, nil
}

func GetPathTail(device string) string {
	strs := strings.Split(device, "/")
	if len(strs) > 0 {
		return strs[len(strs)-1]
	}
	return ""
}

func GetBackendAndVolume(volumeId string) (string, string) {
	var backend, volume string

	splits := strings.SplitN(volumeId, "-", 2)
	if len(splits) == 2 {
		backend, volume = splits[0], splits[1]
	} else {
		backend, volume = "", splits[0]
	}

	log.Infof("Backend %s, volume %s", backend, volume)
	return backend, volume
}

func SplitVolumeId(volumeId string) (string, string) {
	splits := strings.SplitN(volumeId, ".", 2)
	if len(splits) == 2 {
		return splits[0], splits[1]
	}
	return splits[0], ""
}

func MergeMap(args ...map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for _, arg := range args {
		for k, v := range arg {
			newMap[k] = v
		}
	}

	return newMap
}

func WaitUntil(f func() (bool, error), timeout time.Duration, interval time.Duration) error {
	done := make(chan error)

	go func() {
		timeout := time.After(timeout)

		for {
			condition, err := f()
			if err != nil {
				done <- err
				return
			}

			if condition {
				done <- nil
				return
			}

			select {
			case <-timeout:
				done <- fmt.Errorf("Wait timeout")
				return
			default:
				time.Sleep(interval)
			}
		}
	}()

	select {
	case err := <-done:
		return err
	}
}

func RandomInt(n int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(n)
}

func CopyMap(srcMap interface{}) map[string]interface{} {
	copied := make(map[string]interface{})

	if m, ok := srcMap.(map[string]string); ok {
		for k, v := range m {
			copied[k] = v
		}
	} else if m, ok := srcMap.(map[string]interface{}); ok {
		for k, v := range m {
			copied[k] = v
		}
	}

	return copied
}

func StrToBool(str string) bool {
	b, err := strconv.ParseBool(str)
	if err != nil {
		log.Warningf("Parse bool string %s error, return false")
		return false
	}

	return b
}

func ReflectCall(obj interface{}, method string, args ...interface{}) []reflect.Value {
	in := make([]reflect.Value, len(args))
	for i, v := range args {
		in[i] = reflect.ValueOf(v)
	}

	if v := reflect.ValueOf(obj).MethodByName(method); v.IsValid() {
		return v.Call(in)
	}

	return nil
}

func IsSupportClonePair(SystemInfo map[string]interface{}) bool {
	versionInfo := SystemInfo["PRODUCTVERSION"].(string)
	return versionInfo >= DoradoCloneVersion
}

func IsSupportFeature(features map[string]int, feature string) bool {
	var support bool

	status, exist := features[feature]
	if exist {
		support = status == 1 || status == 2
	}

	return support
}
