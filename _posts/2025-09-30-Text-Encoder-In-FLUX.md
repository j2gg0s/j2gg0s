一个典型的文生图模型, 一般包括:
1. Text Encoder, 用提示词(prompt)转化为 embedding 用以指导后续的生图. 模型可能使用多个文本编码器.
2. Scheduler, 包含将初始随机向量逐步去噪生成清晰图片的算法.
3. UNet or Diffusion Transformer(DiT), 模型的核心部分. 在每一步中, 它都会执行去噪预测.
4. Variational Autoencoder(VAE), 将图片编码/解码到潜空间(latent space).

[FLUX.1-dev](https://huggingface.co/black-forest-labs/FLUX.1-dev) 使用了两个文本编码器,
较小的是缘自 OpenAI 的 [CLIP](https://github.com/openai/CLIP), 其负责.
较大的是源自 Google 的 [T5](), 其负责.

我将借助 HuggingFace 的实现, 分析 Text Encoder 的一般逻辑, 并学习 transformer.

## Tokenizer
文本编码的第一步是分词, 即将字符串分割成 token, 再将 token 映射成数字.
后者是因为计算器真正能够识别的仅仅是数字.

以 FLUX.1-dev 使用的 `CLIPTokenizer` 为例,
`Hello World!` 对应的分词结果为 `['hello</w>', 'world</w>', '!</w>']`,
数字结果为 `[49406, 3306, 1002, 256, 49407]`.

抛开此处使用的具体规则, 我们好奇的是: 如何确定分词规则, 如何确定 token 和数字的映射关系.
当前主流的方法是:
准备足够的数据进行训练, 先将其拆分成最基础的 token, 随后以某个标准不断合并 token,
直到 token 的数量少到符合要求后, 将其一一映射到数字.

比如 CLIPTokenizer 使用的 Byte-level Byte-Pair Encoding 训练的主要结果就是:
- [merges.txt](https://huggingface.co/black-forest-labs/FLUX.1-dev/blob/main/tokenizer/merges.txt), 定义了两个连续 token 出现的概率
- [vocab.json](https://huggingface.co/black-forest-labs/FLUX.1-dev/blob/main/tokenizer/vocab.json), 定义了 token 对应的数字.

在分词时，将文本拆分成最基础的 token 后, 不断合并出现频率最高的 token 对,
直到没有预知的 token 对再将其映射到数字.

## Encoder
CLIP 本身是一个用以关联文本和图像的模型, 其使用到的文本编码器通过 transformer 处理分词结果.
