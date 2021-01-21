package schedule

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/boxgo/box/minibox"
	"github.com/boxgo/logger"
	"github.com/boxgo/schedule/lock"
	"github.com/robfig/cron"
)

type (
	// Schedule 定时任务管理
	Schedule struct {
		name        string
		Type        Type        `config:"type" help:"Stop: 0, Once: 1, Timing: 2, OnceAndTiming: 3"`
		LockPrefix  string      `config:"lockPrefix" help:"Prefix of lock"`
		LockSeconds uint        `config:"lockSeconds" help:"Lock ttl"`
		AutoUnlock  bool        `config:"autoUnlock" help:"Auto unlock after task finish"`
		Compete     bool        `config:"compete" help:"Only winner can exec schedule"`
		Spec        string      `config:"spec" help:"Cron spec info"`
		Args        interface{} `config:"args" help:"Args"`
		app         minibox.App

		cron          *cron.Cron
		locker        lock.Lock
		onceHandler   Handler
		timingHandler Handler
		LockDuration  time.Duration
	}

	// Type 定时任务类型
	Type int

	// Handler 任务处理器
	Handler func(sch *Schedule) error
)

const (
	// Stop 停止
	Stop = Type(0)
	// Once 一次性的，立即执行一次
	Once = Type(1)
	// Timing 定时的，周期执行
	Timing = Type(2)
	// OnceAndTiming Once + Timing
	OnceAndTiming = Type(3)
)

// Name 配置名称
func (s *Schedule) Name() string {
	return "schedules." + s.name
}

// ConfigWillLoad 配置文件将要加载
func (s *Schedule) ConfigWillLoad(context.Context) {

}

// ConfigDidLoad 配置文件已经加载。做一些默认值设置
func (s *Schedule) ConfigDidLoad(context.Context) {
	if s.name == "" || s.Spec == "" {
		panic("schedules config is invalid")
	}

	if s.LockPrefix == "" {
		s.LockPrefix = s.app.AppName
	}

	if s.LockSeconds == 0 {
		s.LockDuration = time.Second * 10
	} else {
		s.LockDuration = time.Duration(1000000000 * s.LockSeconds)
	}
}

// Serve schedule
func (s *Schedule) Serve(context.Context) error {
	switch s.Type {
	case Once:
		s.execOnce()
	case Timing:
		s.execTiming()
	case OnceAndTiming:
		s.execOnce()
		s.execTiming()
	}

	return nil
}

// Shutdown stop cron
func (s *Schedule) Shutdown(context.Context) error {
	if s.cron != nil {
		s.cron.Stop()
	}

	return nil
}

// Exts 获取app信息
func (s *Schedule) Exts() []minibox.MiniBox {
	return []minibox.MiniBox{&s.app}
}

// Once 执行一次
func (s *Schedule) execOnce() {
	if s.onceHandler == nil {
		return
	}

	s.exec(s.onceHandler)
}

func (s *Schedule) execTiming() {
	if s.timingHandler == nil {
		return
	}

	c := cron.New()
	c.AddFunc(s.Spec, func() {
		s.exec(s.timingHandler)
	})
	c.Start()

	s.cron = c
}

func (s *Schedule) exec(handler Handler) {
	go func() {
		defer func() {
			if s.Compete && s.AutoUnlock {
				s.locker.UnLock(s.lockKey())
			}

			if err := recover(); err != nil {
				debug.PrintStack()
				logger.Default.Errorf("Schedule [%s] crash: %s", s.name, err)
				return
			}
		}()

		if !s.isWinner() {
			return
		}

		if s.Compete {
			lock, err := s.locker.Lock(s.lockKey(), s.LockDuration)
			if err != nil {
				logger.Default.Errorf("Schedule [%s] lock error: [%s]", s.name, err.Error())
				return
			} else if !lock {
				logger.Default.Errorf("Schedule [%s] lock fail", s.name)
				return
			}
		}

		logger.Default.Infof("Schedule [%s] ready", s.name)

		err := handler(s)
		if err != nil {
			logger.Default.Errorf("Schedule [%s] error: [%s]", s.name, err.Error())
		} else {
			logger.Default.Infof("Schedule [%s] success", s.name)
		}
	}()
}

// compete
// 1. 如果配置为竞争任务，竞争成功才返回true，否则返回false
// 2. 如果配置为非竞争任务，直接返回true
func (s *Schedule) isWinner() bool {
	winner := true

	if s.Compete {
		lcoked, err := s.locker.IsLocked(s.lockKey())
		if err != nil {
			logger.Default.Errorf("Schedule [%s] compete IsLocked error: %#v", s.name, err)
			winner = false
		} else if lcoked {
			logger.Default.Infof("Schedule [%s] compete fail", s.name)
			winner = false
		} else {
			logger.Default.Infof("Schedule [%s] compete success", s.name)
			winner = true
		}
	}

	return winner
}

// lockKey 获取lock的key
func (s *Schedule) lockKey() string {
	return s.LockPrefix + "." + s.Name() + ".locker"
}

// New a schedule
func New(name string, opts ...Option) *Schedule {
	options := &Options{}

	for _, opt := range opts {
		opt(options)
	}

	return &Schedule{
		name:          name,
		locker:        options.locker,
		onceHandler:   options.onceHandler,
		timingHandler: options.timingHandler,
	}
}
