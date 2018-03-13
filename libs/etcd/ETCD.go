package ETCD

import (
	"context"
	"encoding/json"
	"reflect"

	Const "iparking/share/const"

	etcd "github.com/coreos/etcd/clientv3"
)

type ETCDClient struct {
	Client    *etcd.Client
	Context   context.Context
	Configs   map[string]ETCDConfig
	WatchChan chan ETCDConfig
}

func init() {

}

func (this *ETCDClient) Init(conf etcd.Config) error {

	client, err := etcd.New(conf)
	if err != nil {
		return err
	}

	this.Client = client
	this.Context = context.Background()
	this.Configs = make(map[string]ETCDConfig)
	this.WatchChan = make(chan ETCDConfig)

	return nil
}

func (this *ETCDClient) Register(key string, objType reflect.Type) interface{} {

	this.Configs[key] = ETCDConfig{Key: key, Type: objType, Content: nil}

	value, err := this.Refresh(key)
	if err != nil {
		return nil
	}

	return value
}

func (this *ETCDClient) WatchAll() {

	for key, _ := range this.Configs {

		go func(this *ETCDClient, key string) {

			defer func() {
				if e := recover(); e != nil {
				}
			}()

			watchan := this.Client.Watch(this.Context, key)
			for v := range watchan {
				if v.Err() != nil {
					continue
				}

				this.Refresh(key)

				this.WatchChan <- this.Configs[key]
			}

		}(this, key)
	}
}

func (this *ETCDClient) Set(jsonPath string) {

}

func (this *ETCDClient) Get(key string) interface{} {

	config, ok := this.Configs[key]
	if ok == false {
		return nil
	}

	if config.Content != nil {
		return config.Content
	}

	value, err := this.Refresh(key)

	if err != nil {
		return nil
	}

	return value
}

func (this *ETCDClient) Refresh(key string) (interface{}, error) {

	config, ok := this.Configs[key]
	if ok == false {
		return nil, Const.ErrETCD_NotFoundKey
	}

	resp, err := this.Client.Get(context.Background(), key)
	if err != nil || resp.Count <= 0 {
		return nil, err
	}

	obj := reflect.New(config.Type).Interface()
	if err = json.Unmarshal([]byte(resp.Kvs[0].Value), &obj); err != nil {
		return nil, err
	}

	config.Locker.Lock()
	defer config.Locker.Unlock()
	config.Content = obj

	return config.Content, nil
}
