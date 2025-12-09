package docker

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
)

// BuildRequest represents a single image build request
type BuildRequest struct {
	InstanceName string
	BundlePath   string
	BaseImage    string
	ResultChan   chan BuildResult
}

// BuildResult represents the result of a build operation
type BuildResult struct {
	Error error
}

// ImageBuildWorker manages a queue of image build requests
// and processes them with multiple worker goroutines
type ImageBuildWorker struct {
	client      *Client
	jobQueue    chan BuildRequest
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	isShutdown  bool
	mu          sync.RWMutex
	workerCount int
}

// NewImageBuildWorker creates a new image build worker with 3 workers
func NewImageBuildWorker(client *Client) *ImageBuildWorker {
	ctx, cancel := context.WithCancel(client.ctx)
	return &ImageBuildWorker{
		client:      client,
		jobQueue:    make(chan BuildRequest, 100), // Buffer for up to 100 requests
		ctx:         ctx,
		cancel:      cancel,
		workerCount: 3, // 3 concurrent workers
	}
}

// Start begins the worker goroutines that process build requests
func (w *ImageBuildWorker) Start() {
	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.worker(i + 1)
	}
	logrus.Infof("Started %d image build workers", w.workerCount)
}

// worker is a goroutine that processes build requests from the queue
func (w *ImageBuildWorker) worker(id int) {
	defer w.wg.Done()

	logrus.Infof("Image build worker #%d started", id)

	for {
		select {
		case <-w.ctx.Done():
			logrus.Infof("Image build worker #%d shutting down", id)
			return
		case req := <-w.jobQueue:
			logrus.Infof("Worker #%d processing build request for %s", id, req.InstanceName)
			w.processBuildRequest(req)
		}
	}
}

// processBuildRequest handles a single build request
func (w *ImageBuildWorker) processBuildRequest(req BuildRequest) {
	logrus.WithFields(logrus.Fields{
		"instanceName": req.InstanceName,
		"bundlePath":   req.BundlePath,
	}).Info("Processing image build request")

	err := w.buildImage(req.InstanceName, req.BundlePath, req.BaseImage)

	// Send result back through the channel
	req.ResultChan <- BuildResult{Error: err}
	close(req.ResultChan)

	if err != nil {
		logrus.WithError(err).WithField("instanceName", req.InstanceName).Error("Image build failed")
	} else {
		logrus.WithField("instanceName", req.InstanceName).Info("Image build completed successfully")
	}
}

// buildImage performs the actual image build operation
func (w *ImageBuildWorker) buildImage(instanceName string, bundlePath string, baseImage string) error {
	imageName := fmt.Sprintf("%s:%s", simCliPrefix, instanceName)
	contextTar, err := BuildContextTar(bundlePath, baseImage)
	if err != nil {
		return err
	}

	imageBuildResponse, err := w.client.APIClient.ImageBuild(w.client.ctx, bytes.NewReader(contextTar.Bytes()), types.ImageBuildOptions{
		Tags: []string{imageName},
		Labels: map[string]string{
			bundleNameKey: instanceName,
		},
	})

	if err != nil {
		return err
	}

	return readResponse(imageBuildResponse.Body)
}

// SubmitBuildRequest submits a build request and waits for the result
// This method blocks until the build is complete
func (w *ImageBuildWorker) SubmitBuildRequest(instanceName string, bundlePath string, baseImage string) error {
	w.mu.RLock()
	if w.isShutdown {
		w.mu.RUnlock()
		return fmt.Errorf("worker is shutdown")
	}
	w.mu.RUnlock()

	resultChan := make(chan BuildResult, 1)
	req := BuildRequest{
		InstanceName: instanceName,
		BundlePath:   bundlePath,
		BaseImage:    baseImage,
		ResultChan:   resultChan,
	}

	logrus.WithField("instanceName", instanceName).Info("Submitting image build request to queue")

	// Send the request to the worker
	select {
	case w.jobQueue <- req:
		// Request submitted successfully
	case <-w.ctx.Done():
		return fmt.Errorf("worker context cancelled")
	}

	// Wait for the result
	result := <-resultChan
	return result.Error
}

// Shutdown gracefully shuts down the worker
func (w *ImageBuildWorker) Shutdown() {
	w.mu.Lock()
	w.isShutdown = true
	w.mu.Unlock()

	logrus.Info("Shutting down image build workers")
	w.cancel()
	w.wg.Wait()
	logrus.Info("All image build workers stopped")
}
