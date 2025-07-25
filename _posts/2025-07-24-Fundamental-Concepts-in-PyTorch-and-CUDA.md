## Driver 和 CUDA
Driver, 驱动, 是 NVIDIA 提供给操作系统识别并管理硬件的软件.

CUDA Toolkit 是 NVIDIA 提供给开发者调用 GPU 的高级抽象方案.
我们可以简单的将其理解为:
- 一套 API, 针对 GPU 功能的高级抽象.
- 一个编译器, nvcc, 将开发者调用上述 API 的代码编译成驱动和操作系统能够识别的指令.
  项目中传统的 C/C++ 代码, nvcc 会将其转交给 GCC/CLang 等传统编辑器处理.
- 一套运行时, libcuda/licudart, 需要提前将这部分 lib 部署到运行的系统中.

做为一个上层使用者, 我们大多数时候只需要关注 CUDA 的版本.

## PyTorch
你可以将 PyTorch 理解为在 CUDA 基础上针对深度神级网络的进一步抽象和封装.
当然实际上 PyTorch 不仅仅支持 NVIDIA GPU, 也支持其他公司的显卡, 甚至 CPU.
其内部通过 C++, 针对不同的设备, 抽象出了一套统一的接口,
在此之上通过 Python 再封装了一层 Pythonic 的易用 API.

所以 PyTorch 本身即包括了需要编译的 C++ 代码, 也包括了解释型的 Python 代码.
我们日常使用时, 可以直接通过 pip 安装 PyTorch 预先针对不同操作系统,
不同 CUDA 版本预先编译好的二进制包, [Installing previous versions of PyTorch](https://pytorch.org/get-started/previous-versions/).

## For k8s
针对生产环境, 我们会更进一步, 选择在 NVIDIA/PyTorch 提供的镜像的基础上构建我们自己的项目.
比如说 [nvidia/cuda:12.1.1-runtime-ubuntu22.04](https://hub.docker.com/layers/nvidia/cuda/12.1.1-runtime-ubuntu22.04/images/sha256-2541299cf78eee8ee2b415782ad6f083bc4d351c4e40f104c776159bd483650e),
就是在 Ubuntu 22.04 中预先装好 CUDA 12.1.1 的运行时, 我们随后只需要通过 pip install pytorch 就可以.

为了在 k8s 中运行我们的镜像, 我们还需要使得镜像可以识别并调用 GPU.
大多数情况下我们依赖预先安装在集群内的 nvidia-container-runtime 将宿主机上的驱动和
GPU 映射到容器内.
