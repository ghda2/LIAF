package web

import (
	"io/fs"
	"os"
	"sync"
	"time"
)

const DefaultMaxRAMAssetSize = 512 * 1024 // 512 KB: arquivos acima disso vao para disco/streaming

type CachedAsset struct {
	Path        string
	ContentType string
	Content     []byte
	GzipContent []byte
	ETag        string
	Size        int
	DiskPath    string
	IsDisk      bool
	ModTime     time.Time
}

type AssetCache struct {
	mu              sync.RWMutex
	assets          map[string]*CachedAsset
	MaxRAMAssetSize int64
}

var (
	embeddedMu sync.RWMutex
	embeddedFS fs.FS
)

// SetEmbeddedFS registra um sistema de arquivos virtual embutido (ex: embed.FS) para Single-Binary deploy.
func SetEmbeddedFS(fileSys fs.FS) {
	embeddedMu.Lock()
	defer embeddedMu.Unlock()
	embeddedFS = fileSys
}

func GetEmbeddedFS() fs.FS {
	embeddedMu.RLock()
	defer embeddedMu.RUnlock()
	return embeddedFS
}

func NewAssetCache() *AssetCache {
	return &AssetCache{
		assets:          make(map[string]*CachedAsset),
		MaxRAMAssetSize: DefaultMaxRAMAssetSize,
	}
}

// LoadDirectory carrega os assets a partir de um diretório em disco no sandbox os.Root.
func (ac *AssetCache) LoadDirectory(dirPath string) error {
	if efs := GetEmbeddedFS(); efs != nil {
		return ac.LoadFS(efs)
	}

	root, err := os.OpenRoot(dirPath)
	if err != nil {
		return err
	}
	defer root.Close()

	ac.mu.Lock()
	defer ac.mu.Unlock()

	return loadPipeline(ac, root.FS(), dirPath)
}

// LoadFS carrega todos os assets a partir de um sistema de arquivos virtual (ex: embed.FS).
func (ac *AssetCache) LoadFS(fsys fs.FS) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	return loadPipeline(ac, fsys, "")
}

func (ac *AssetCache) Get(route string) (*CachedAsset, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	asset, ok := ac.assets[route]
	return asset, ok
}

func (ac *AssetCache) Count() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return len(ac.assets)
}
