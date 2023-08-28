[Stable Diffusion][] 是著名的 text-to-image 模型, 可以直接基于文字描述生成对应图片.

## Diffusion Model
Diffusion Model 从属于深度学习领域,
对于具体输入, 能够基于训练的数据, 生成类似结果.

Forward diffusion 是训练的过程, 对于给定的图片, 通过不断添加噪音, 得到完全不可识别的随机图片.
Reverse diffusion 是推理的过程, 针对特定输入, 对随机的图片, 通过不断去除噪音, 生成符合输入的图片.

上述的核心U-Net, 卷积神经网络(convolutional neural network, CNN) 的一种实现, 属于超纲内容, 我发现我可能无法在短时间内理解它.
Stable Diffusion 中将这部分称作 noise predictor.

## Latent
相较于文本, 图片的信息量是巨大的.
一张 512*512 像素, 三原色(Red, Green, Blue) 的图片也含有 786,432 中可能.
我们基本没有办法在个人设备上, 直接在这个纬度(pixel space) 运行 diffusion model.

[Stable Diffusion][] 通过变分自编码器(variational autoencoder, VAE), 将图片从 pixel space 压缩到 latent space, 缩小了两位数的运算量.

VAE 在 SD 中可行的主要原因是真实世界的图片中的各个像素具有高度的规律性和相关性.
比如说人脸总是应该有嘴巴, 鼻子和眼睛. 所以通过 VAE 压缩图片并不会损失太多的信息.

## Text Prompt
模型并不能直接理解文本, 需要:
- 首先通过分词器(tokenizer) 将文本分割成 token.
- 再通过 embedding 将 token 映射到向量, embedding 来自预先的训练.
- 最后向量经有 transformer 转换后输出到 predictor

## Attention
Attention 是 Transformer 的著名机制.

Self attention 能够将输入流中的 token 关联起来.
比如对于输入 "Bark is very cute and he is a dog",
经过 self-attention 机制处理后, 模型就会将 Bark 和 he 基本等价.

相较于 self attention, cross attention 是针对两个输入流的类似机制.
SD 中就通过 cross attention 将 text prompt 和图片输入关联起来.

## .ckpt 和 .safetensors
ckpt 和 safetensors 都是 SD 相关的模型文件, safetensors 相对 ckpt 可以避免恶意代码的注入.

那么模型文件是什么呢? 从 pytorch 的文档来看很直接, 就是 python 的类名和相关参数.
sd-webui 读取模型文件后, 从 target 字段读取到对应的类名, 结合 params 字段初始化,
随后按照 torch 的习惯从模型文件中加载 state_dict.

SD 1.5 使用的默认模型 [v1-5-pruned-emaonly.safetensors](https://huggingface.co/runwayml/stable-diffusion-v1-5/resolve/main/v1-5-pruned-emaonly.safetensors)
对应类即为 [ldm.models.diffusion.ddpm.LatentDiffusion](https://github.com/CompVis/stable-diffusion/blob/main/ldm/models/diffusion/ddpm.py#L424)
