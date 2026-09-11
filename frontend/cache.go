package frontend

import (
	"os"
	"strings"
)

// CacheDescriptor names one downloadable artifact, so that a cache resolver can
// decide where its bytes live on disk.
type CacheDescriptor struct {
	name       string
	version    string
	classifier string
	extension  string
}

// NewCacheDescriptor describes an artifact that needs no classifier.
func NewCacheDescriptor(name, version, extension string) *CacheDescriptor {
	return NewClassifiedCacheDescriptor(name, version, "", extension)
}

// NewClassifiedCacheDescriptor describes an artifact whose identity includes a
// platform classifier, as a Node.js distribution's does.
func NewClassifiedCacheDescriptor(name, version, classifier, extension string) *CacheDescriptor {
	return &CacheDescriptor{
		name:       name,
		version:    version,
		classifier: classifier,
		extension:  extension,
	}
}

// Name is the artifact name, such as "node" or "yarn".
func (d *CacheDescriptor) Name() string { return d.name }

// Version is the artifact version, as configured.
func (d *CacheDescriptor) Version() string { return d.version }

// Classifier is the platform classifier, empty when the artifact has none.
func (d *CacheDescriptor) Classifier() string { return d.classifier }

// Extension is the archive extension, such as "tar.gz".
func (d *CacheDescriptor) Extension() string { return d.extension }

// CacheResolver decides where a downloaded artifact is kept between builds.
type CacheResolver interface {
	Resolve(descriptor *CacheDescriptor) string
}

// DirectoryCacheResolver keeps every artifact in one flat directory, named
// "<name>-<version>[-<classifier>].<extension>".
type DirectoryCacheResolver struct {
	cacheDirectory string
}

// NewDirectoryCacheResolver caches into cacheDirectory, which is created on
// first use.
func NewDirectoryCacheResolver(cacheDirectory string) *DirectoryCacheResolver {
	return &DirectoryCacheResolver{cacheDirectory: cacheDirectory}
}

// Resolve returns the cache path for a descriptor, creating the cache directory
// if it is not there yet.
func (r *DirectoryCacheResolver) Resolve(descriptor *CacheDescriptor) string {
	if !exists(r.cacheDirectory) {
		_ = os.MkdirAll(r.cacheDirectory, 0o777)
	}

	var filename strings.Builder
	filename.WriteString(descriptor.Name())
	filename.WriteString("-")
	filename.WriteString(descriptor.Version())
	if descriptor.Classifier() != "" {
		filename.WriteString("-")
		filename.WriteString(descriptor.Classifier())
	}
	filename.WriteString(".")
	filename.WriteString(descriptor.Extension())
	return childFile(r.cacheDirectory, filename.String())
}
