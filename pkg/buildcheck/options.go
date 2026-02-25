package buildcheck

type Option func(*Detector)

func WithHasher(hasher HashComputer) Option {
	return func(d *Detector) {
		d.hasher = hasher
	}
}

type FileStoreOption func(*FileStore)

func WithFileStoreBaseDir(dir string) FileStoreOption {
	return func(fs *FileStore) {
		fs.baseDir = dir
	}
}
