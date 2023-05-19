package utils

import (
	"context"
	"strconv"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
)

var (
	// configIsInit = false

	configIsBatch      = false
	configBatchCount   = 1
	configIsBlockingUp = true

	configIsBatchIsReady      = false
	configBatchCountReady     = false
	configIsBlockingUpIsReady = false
)

// ConfigRun is used to listen for changes to config
func ConfigRun() {
	go func() {
		go configInit()
		logger := log.FromContext(context.Background())
		for {
			select {
			case configIsBatch = <-ConfigIsBatchChannel:
				logger.Info("configIsBatch is :" + strconv.FormatBool(configIsBatch))
				if !configIsBatchIsReady {
					logger.Info("configIsBatch is ready:")
					configIsBatchIsReady = true
				}
			case configBatchCount = <-ConfigBatchCountChannel:
				logger.Info("configBatchCount is :" + strconv.Itoa(configBatchCount))
				if !configBatchCountReady {
					logger.Info("configBatchCount is ready")
					configBatchCountReady = true
				}
			case configIsBlockingUp = <-ConfigIsBlockingUpChannel:
				logger.Info("configIsBlockingUp is :" + strconv.FormatBool(configIsBlockingUp))
				if !configIsBlockingUpIsReady {
					logger.Info("configIsBlockingUp is ready")
					configIsBlockingUpIsReady = true
				}
			}
		}
	}()
}

// configInit is used to initialize config
func configInit() {
	for {
		newBatchConfig()
		newBlockUpConfig()
		time.Sleep(time.Second)
	}
}

// newBatchConfig is used to initialize config
func newBatchConfig() {
	opsConfigInfo := v1alpha1.OpsConfigInfo{}
	err := App.Client.Get(context.Background(), client.ObjectKey{Name: ConfigNameIsBranch, Namespace: AlterShieldOperatorNamespace}, &opsConfigInfo)
	if err != nil {
		if errors.IsNotFound(err) {
			newRecord := NewOpsConfigInfoBatchFunc()
			err := App.Client.Create(context.Background(), newRecord)
			if err != nil {
				log.FromContext(context.Background()).Error(err, "newBatchConfig:create batch config error")
			}
		} else {
			log.FromContext(context.Background()).Error(err, "newBatchConfig:get batch config error")
		}
	}
}

// newBlockUpConfig is used to initialize config
func newBlockUpConfig() {
	opsConfigInfo := v1alpha1.OpsConfigInfo{}
	err := App.Client.Get(context.Background(), client.ObjectKey{Name: ConfigNameIsBlockingUp, Namespace: AlterShieldOperatorNamespace}, &opsConfigInfo)
	if err != nil {
		if errors.IsNotFound(err) {
			newRecord := NewOpsConfigInfoBlockFunc()
			err := App.Client.Create(context.Background(), newRecord)
			if err != nil {
				log.FromContext(context.Background()).Error(err, "newBlockUpConfig:create Block config error")
			}
		} else {
			log.FromContext(context.Background()).Error(err, "newBlockUpConfig:get Block config error")
		}
	}
}

// ConfigIsBatch It is guaranteed to be called when configIsBatchIsReady is true
func ConfigIsBatch() bool {
	// Without volatile, using lock guarantees visibility, has an impact on performance, and can be modified if there is a better way
	mutex := &sync.Mutex{}
	mutex.Lock()
	if configIsBatchIsReady {
		defer mutex.Unlock()
		return configIsBatch
	} else {
		defer mutex.Unlock()
		time.Sleep(time.Second)
		return ConfigIsBatch()
	}
}

// ConfigBatchCount It is guaranteed to be called when configBatchCountReady is true
func ConfigBatchCount() int {
	// Without volatile, using lock guarantees visibility, has an impact on performance, and can be modified if there is a better way
	mutex := &sync.Mutex{}
	mutex.Lock()
	if configBatchCountReady {
		defer mutex.Unlock()
		return configBatchCount
	} else {
		defer mutex.Unlock()
		time.Sleep(time.Second)
		return ConfigBatchCount()
	}
}

// ConfigIsBlockingUp It is guaranteed to be called when configIsBlockingUpIsReady is true
func ConfigIsBlockingUp() bool {
	// Without volatile, using lock guarantees visibility, has an impact on performance, and can be modified if there is a better way
	mutex := &sync.Mutex{}
	mutex.Lock()
	if configIsBlockingUpIsReady {
		defer mutex.Unlock()
		return configIsBlockingUp
	} else {
		defer mutex.Unlock()
		time.Sleep(time.Second)
		return ConfigIsBlockingUp()
	}
}
