package cache

import (
	"time"

	"github.com/spf13/afero"
)

func Cache(base afero.Fs, root string, ttl time.Duration, cacheDir string) afero.Fs {
	return afero.NewCacheOnReadFs(afero.NewBasePathFs(base, root), afero.NewBasePathFs(afero.NewOsFs(), cacheDir), ttl)
}
