package buildcheck

import (
	"fmt"
	"time"
)

type Detector struct {
	store  ManifestStore
	hasher HashComputer
}

func NewDetector(store ManifestStore, opts ...Option) *Detector {
	d := &Detector{
		store:  store,
		hasher: NewSHA256Hasher(),
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

func (d *Detector) DetectChanges(config ImageConfig) (*ChangeReport, error) {
	oldManifest, err := d.store.Load(config.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	newFileHashes, newSourceHash, err := d.hasher.ComputeDirectoryHash(
		config.ContextPath,
		config.IgnorePaths,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to compute source hash: %w", err)
	}

	report := &ChangeReport{
		ImageName:  config.Name,
		ComputedAt: time.Now(),
		SourceHash: newSourceHash,
		Changes:    []Change{},
	}

	if oldManifest == nil {
		report.HasChanges = true
		report.NewManifest = &Manifest{
			ImageName:  config.Name,
			SourceHash: newSourceHash,
			FileHashes: newFileHashes,
		}
		for path, fh := range newFileHashes {
			report.Changes = append(report.Changes, Change{
				Type:    ChangeTypeAdded,
				Path:    path,
				NewHash: fh.Hash,
			})
		}
		return report, nil
	}

	report.OldManifest = oldManifest

	newManifest := &Manifest{
		ImageName:  config.Name,
		SourceHash: newSourceHash,
		FileHashes: newFileHashes,
	}
	report.NewManifest = newManifest

	for path, oldFH := range oldManifest.FileHashes {
		if newFH, exists := newFileHashes[path]; exists {
			if oldFH.Hash != newFH.Hash {
				report.Changes = append(report.Changes, Change{
					Type:    ChangeTypeModified,
					Path:    path,
					OldHash: oldFH.Hash,
					NewHash: newFH.Hash,
				})
			}
		} else {
			report.Changes = append(report.Changes, Change{
				Type:    ChangeTypeDeleted,
				Path:    path,
				OldHash: oldFH.Hash,
			})
		}
	}

	for path, newFH := range newFileHashes {
		if _, exists := oldManifest.FileHashes[path]; !exists {
			report.Changes = append(report.Changes, Change{
				Type:    ChangeTypeAdded,
				Path:    path,
				NewHash: newFH.Hash,
			})
		}
	}

	report.HasChanges = len(report.Changes) > 0
	return report, nil
}

func (d *Detector) ComputeSourceHash(config ImageConfig) (string, map[string]FileHash, error) {
	fileHashes, sourceHash, err := d.hasher.ComputeDirectoryHash(config.ContextPath, config.IgnorePaths)
	return sourceHash, fileHashes, err
}

func (d *Detector) NeedsRebuild(config ImageConfig) (bool, *ChangeReport, error) {
	report, err := d.DetectChanges(config)
	if err != nil {
		return true, nil, err
	}
	return report.HasChanges, report, nil
}

func (d *Detector) RecordBuild(config ImageConfig, tag string) error {
	manifest, err := d.store.Load(config.Name)
	if err != nil {
		return err
	}

	if manifest == nil {
		fileHashes, sourceHash, err := d.hasher.ComputeDirectoryHash(config.ContextPath, config.IgnorePaths)
		if err != nil {
			return err
		}
		manifest = &Manifest{
			ImageName:  config.Name,
			SourceHash: sourceHash,
			FileHashes: fileHashes,
		}
	}

	manifest.LastBuildAt = time.Now()
	manifest.LastBuildTag = tag
	manifest.BuildCommand = config.BuildCommand
	manifest.BuildArgs = config.BuildArgs

	return d.store.Save(manifest)
}
