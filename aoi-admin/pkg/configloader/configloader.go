package configloader

import (
	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Event 描述配置文件变化事件，避免上层直接依赖 fsnotify 的具体类型。
type Event struct {
	Name string
	Op   string
}

// Loader 用项目自有类型封装配置解析和文件监听，便于替换底层配置库。
type Loader struct {
	v *viper.Viper
}

// New 创建隔离的配置加载器，避免不同调用方共享 viper 全局状态。
func New() *Loader {
	return &Loader{v: viper.New()}
}

// LoadEnv 尝试加载 dotenv 文件；缺失或加载失败时交给后续配置校验暴露问题。
func LoadEnv(path string) {
	_ = godotenv.Load(path)
}

func (l *Loader) SetConfigFile(path string) {
	l.v.SetConfigFile(path)
}

func (l *Loader) ReadInConfig() error {
	return l.v.ReadInConfig()
}

func (l *Loader) AllSettings() map[string]any {
	return l.v.AllSettings()
}

func (l *Loader) Set(key string, value any) {
	l.v.Set(key, value)
}

func (l *Loader) Unmarshal(out any) error {
	return l.v.Unmarshal(out)
}

// OnConfigChange 将 fsnotify 事件转换为项目内部事件，减少配置监听的外部耦合。
func (l *Loader) OnConfigChange(handler func(Event)) {
	l.v.OnConfigChange(func(e fsnotify.Event) {
		handler(Event{Name: e.Name, Op: e.Op.String()})
	})
}

func (l *Loader) WatchConfig() {
	l.v.WatchConfig()
}
