package volume

import (
	"errors"
	"fmt"

	log "github.com/golang/glog"

	"github.com/Huawei/eSDK_K8S_Plugin/pkg/storage/fusionstorage/client"
	"github.com/Huawei/eSDK_K8S_Plugin/pkg/utils"
	"github.com/Huawei/eSDK_K8S_Plugin/pkg/utils/taskflow"
)

type SAN struct {
	cli *client.Client
}

func NewSAN(cli *client.Client) *SAN {
	return &SAN{
		cli: cli,
	}
}

func (p *SAN) preCreate(params map[string]interface{}) error {
	name := params["name"].(string)
	params["name"] = utils.GetFusionStorageLunName(name)

	if v, exist := params["storagepool"].(string); exist {
		pool, err := p.cli.GetPoolByName(v)
		if err != nil {
			return err
		}
		if pool == nil {
			return fmt.Errorf("Storage pool %s doesn't exist", v)
		}

		params["poolId"] = int64(pool["poolId"].(float64))
	}

	if v, exist := params["clonefrom"].(string); exist && v != "" {
		params["clonefrom"] = utils.GetFusionStorageLunName(v)
	}

	return nil
}

func (p *SAN) Create(params map[string]interface{}) error {
	err := p.preCreate(params)
	if err != nil {
		return err
	}

	taskflow := taskflow.NewTaskFlow("Create-FusionStorage-LUN-Volume")
	taskflow.AddTask("Create-LUN", p.createLun, nil)

	err = taskflow.Run(params)
	if err != nil {
		taskflow.Revert()
		return err
	}

	return nil
}

func (p *SAN) createLun(params, taskResult map[string]interface{}) (map[string]interface{}, error) {
	name := params["name"].(string)

	vol, err := p.cli.GetVolumeByName(name)
	if err != nil {
		log.Errorf("Get LUN %s error: %v", name, err)
		return nil, err
	}

	if vol == nil {
		_, exist := params["clonefrom"]
		if exist {
			err = p.clone(params)
		} else {
			err = p.cli.CreateVolume(params)
		}
	}

	if err != nil {
		log.Errorf("Create LUN %s error: %v", name, err)
		return nil, err
	}

	return nil, nil
}

func (p *SAN) clone(params map[string]interface{}) error {
	cloneFrom := params["clonefrom"].(string)

	srcVol, err := p.cli.GetVolumeByName(cloneFrom)
	if err != nil {
		log.Errorf("Get clone src vol %s error: %v", cloneFrom, err)
		return err
	}
	if srcVol == nil {
		msg := fmt.Sprintf("Clone src vol %s does not exist", cloneFrom)
		log.Errorln(msg)
		return errors.New(msg)
	}

	volCapacity := params["capacity"].(int64)
	if volCapacity < int64(srcVol["volSize"].(float64)) {
		msg := fmt.Sprintf("Clone vol capacity must be >= src %s", cloneFrom)
		log.Errorln(msg)
		return errors.New(msg)
	}

	snapshotName := fmt.Sprintf("k8s_vol_%s_snap_%d", cloneFrom, utils.RandomInt(10000000000))

	err = p.cli.CreateSnapshot(snapshotName, cloneFrom)
	if err != nil {
		log.Errorf("Create snapshot %s error: %v", snapshotName, err)
		return err
	}

	defer func() {
		p.cli.DeleteSnapshot(snapshotName)
	}()

	volName := params["name"].(string)

	err = p.cli.CreateVolumeFromSnapshot(volName, volCapacity, snapshotName)
	if err != nil {
		log.Errorf("Create volume %s from %s error: %v", volName, snapshotName, err)
		return err
	}

	return nil
}

func (p *SAN) Delete(name string) error {
	vol, err := p.cli.GetVolumeByName(name)
	if err != nil {
		log.Errorf("Get volume by name %s error: %v", name, err)
		return err
	}
	if vol == nil {
		log.Warningf("Volume %s doesn't exist while trying to delete it", name)
		return nil
	}

	return p.cli.DeleteVolume(name)
}
