package delayQS

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Config struct {
	Name string
}

func InitConf(cfgName string ) error {
	c := Config{
		Name:cfgName,
	}
	 if err := c.initConfig(); err != nil{
	 	return  nil
	 }
	 c.watchConfig()

	 return nil

}

//读取配置文件
func (c *Config ) initConfig() error {
	if c.Name != "" {
		viper.SetConfigFile( c.Name )
	}else{
		viper.AddConfigPath("conf")
		viper.SetConfigName("config")
	}

	//设置配置文件的格式
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		return  err
	}

	return  nil
}

//监听文件
func (c *Config) watchConfig() error {
	viper.WatchConfig()
	viper.OnConfigChange(func( e fsnotify.Event ){
		fmt.Printf("Config file changed: %s\n", e.Name)
	})
	return  nil
}