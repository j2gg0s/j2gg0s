我们在 k8s 中部署了 [stable-diffusion-webui](https://github.com/AUTOMATIC1111/stable-diffusion-webui)
供任何想要体验的 Stable Diffusion Model 的用户使用.
随着一个又一个的请求, 我们频繁的遇到 CUDA 的 OOM 错误.
其中的一小部分确实是因为用户请求需要的资源超过了对应 GPU 能够提供的内存.

剩下的, 占大部分的, 是类似如下的令人困惑的场景.
```json
{"error": "OutOfMemoryError", "detail": "", "body": "", "errors": "CUDA out of memory. Tried to allocate 1024.00 MiB (GPU 0; 11.76 GiB total capacity; 7.92 GiB already allocated; 784.31 MiB free; 10.63 GiB reserved in total by PyTorch) If reserved memory is >> allocated memory try setting max_split_size_mb to avoid fragmentation.  See documentation for Memory Management and PYTORCH_CUDA_ALLOC_CONF"}
```
根据对 [memory_stats](https://pytorch.org/docs/stable/generated/torch.cuda.memory_stats.html#torch.cuda.memory_stats) 的理解:
- GPU 的内存是 11.76G
- pytorch 已经从 GPU 出请求的内存是 10.63G
- pytorch 已经分配给用户的内存是 7.92G
- pytorch 还可以分配的内存为 784.31M, 远小于 reserved 减 allocated 的 2.71G

这部分内存去哪儿了呢? 为什么在用户申请的时候依然没有被回收呢?

## pytorch 是如何分配内存的?.
当用户请求内存时, pytorch 的处理流程可以简化为:
1. 尝试通过 `get_free_block` 去寻找满足要求的空闲 Block
2. 如果失败, 则通过 `trigger_free_memory_callbacks` 去回收已分配但不再使用的 Block 后, 再次尝试 `get_free_block`
3. 如果失败, 则通过 `alloc_block` 去向 GPU 申请新的 Block
4. 如果失败, 则通过 `release_available_cached_blocks` 将已申请但未分配的 Block 释放后再次尝试 `alloc_block`
5. 如果失败, 则通过 `release_cached_blocks` 将所有已申请但未分配的 Block 释放, 再次尝试 `alloc_block`

我们注意到 pytorch 向 GPU 申请和分配给用户的内存都以 Block 为单位.
pytorch 向 GPU 申请的 Block 大小并不固定, 受当时用户请求内存大小的影响.
用户释放内存后, Block 返回给 pytorch 并成为空闲状态.
用户下次申请时优先会复用空闲 Block, 而不是直接向 GPU 申请.

如果用户申请的内存大小小于满足要求的空闲 Block, pytorch 会进行一次 split 操作.
将 Block 分割成两个 Block, 除去用户请求大小的内存会被分割成一个独立的 Block,
留待后用并通过双向链表和分配给用户的 Block 相关联.

`trigger_free_memory_callbacks` 的回收过程会将相邻的空闲 Block 合并, 提高后续分配的灵活性.

相较于其他内存管理机制, pytorch 的内存管理相对简略:
- pytorch 回收 Block, 只尝试合并相邻的空闲的 Block, 并不会进行搬运操作来处理不相连的空闲 Block
- 一旦 Block 被分割, 则 pytorch 无法将其释放. cudaMalloc 和 cudaFree 是对称的, 你无法仅释放某次分配的一部分内存.

上述的两点, 造成了 pytorch 可能因为 Block 碎片化, 导致大量内存无法被使用.

假设在某次分配内存时, pytorch 根据用户请求向 GPU 申请了一个 256M 的 Block.\
<-------------------------- 256M ----------------------------->

经过多次分配和回收, 其使用情况可能变成如下.\
<-- 28M(allocated) --><-- 100M(free) --><-- 28M(allocated) --><-- 100M(free) -->

此时如果用户申请 160M 内存:
- 虽然空闲的总内存大于 160M, 但是因为没有大于 160M 的 Block, 所以无法分配
- pytorch 也无法将空闲的 100M+100M 内存返回给 GPU, 导致也无法向 GPU 申请 160M 内存.

## max_split_size_mb 的作用
max_split_size_mb 的作用在于禁止 pytorch 对任何大于该大小的 Block 进行分割操作, 从而控制碎片化的程度.
我们上文讲诉的都是在未主动设置 max_split_size_mb 的情况下的逻辑, 此时 max_split_size_mb 取默认值 MAX_INT.

我们并没有找到官方推荐的 max_split_size_mb, 我们也不熟悉 pytorch 和 nvida, 很难给出一个很好的推荐值.
从实际使用来和直观逻辑来说, 128/256/512 之类的值都是可选的, 切实的避免了 OOM, 也没有导致明显的性能负担.

## garbage_collection_threshold
pytorch 默认仅在无法获取到合适的空闲 Block 时触发回收,
这个值可以控制当 allocated/capacity 超过此值时触发主动的回收.

## Expandable Segments
pytorch 最新(>v2.0.1)的master分支中添加了 [Expandable Segments](https://github.com/pytorch/pytorch/blob/main/c10/cuda/CUDACachingAllocator.cpp#L267),
可能也可以缓解碎片化的问题.

## References
- [Basic things you might not know: How to avoid CUDA OUT OF MEMORY?](https://civitai.com/articles/194/basic-things-you-might-not-know-how-to-avoid-cuda-out-of-memory)
- [通过设置PYTORCH_CUDA_ALLOC_CONF中的max_split_size_mb解决Pytorch的显存碎片化导致的CUDA:Out Of Memory问题](https://blog.csdn.net/MirageTanker/article/details/127998036)
- [一文读懂 PyTorch 显存管理机制](https://zhuanlan.zhihu.com/p/486360176)
- [pytorch's Memory management](https://pytorch.org/docs/stable/notes/cuda.html#memory-management)
