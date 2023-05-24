package src

import (
	"sync"
	"time"
)

func ProcessPill(pillMx *sync.Mutex, enemiesStatusMx *sync.RWMutex, enemies *[]*Enemy, pillTimer *time.Timer, cfg *Config) {
	pillMx.Lock()
	go UpdateEnemies(enemies, EnemyStatusBlue, enemiesStatusMx)
	if pillTimer != nil {
		pillTimer.Stop()
	}
	pillTimer = time.NewTimer(time.Second * cfg.PillDurationSecs)
	pillMx.Unlock()
	<-pillTimer.C
	pillMx.Lock()
	pillTimer.Stop()
	go UpdateEnemies(enemies, EnemyStatusNormal, enemiesStatusMx)
	pillMx.Unlock()
}
