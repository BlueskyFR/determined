package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage = "mackrorysd/environments:py-3.7-pytorch-1.9-lightning-1.3-tf-2.4-cpu-0.16.2"
	GPUImage = "mackrorysd/environments:cuda-11.1-pytorch-1.9-lightning-1.3-tf-2.4-gpu-0.16.2"
)
