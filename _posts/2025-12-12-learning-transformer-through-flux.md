# 通过 FLUX 学习 Transformer

当进入一个全新的领域, 我们会选择一个典型, 先忽略细节, 构建一个整体的认知, 随后再逐层推进.

### FLUX
以这种方式来认识文生图时:
- 模型: FLUX, 开源模型中的重要玩家
- 实现: diffusers, 足够流行的同时代码质量高, 不会给理解文生图带来额外的负担

以最粗糙的方式来理解 FLUX 的话, 其包括:
- Text Encoder, 文本编码器, 将用户的提示词(prompt)编码为张量(tensor).
- Transformer, 通过 `scheduler` 控制的多轮迭代, 参照用户的提示词生成最终的潜变量.
- Image Decoder, 图像解码器, 将模型生成的变量解码成图片.

这是 diffusers 使用的 `model_index.json`:
```json
{
  "_class_name": "FluxPipeline",
  "_diffusers_version": "0.30.0.dev0",
  "scheduler": [
    "diffusers",
    "FlowMatchEulerDiscreteScheduler"
  ],
  "text_encoder": [
    "transformers",
    "CLIPTextModel"
  ],
  "text_encoder_2": [
    "transformers",
    "T5EncoderModel"
  ],
  "tokenizer": [
    "transformers",
    "CLIPTokenizer"
  ],
  "tokenizer_2": [
    "transformers",
    "T5TokenizerFast"
  ],
  "transformer": [
    "diffusers",
    "FluxTransformer2DModel"
  ],
  "vae": [
    "diffusers",
    "AutoencoderKL"
  ]
}
```

用于图像解码的是 `AutoencoderKL`, 是一个 Variational Auto-Encoder (VAE) 模型.
其作用是将图片编码到潜空间的张量, 或者从潜空间的张量中解码出图片.
其原理是基于构建的神经网络，利用海量的图片, 训练编码和解码, 学习参数.

上文提到的潜空间(latent space)是一个专有的名词, 
是指图片的像素空间经过 VAE 压缩后的产物.
一张 `1024*1024` 的 RGB 图片在像素空间需要使用 `1024*1024*3` 个元素来存储,
在 FLUX 中经过 VAE 压缩后变为 `128*128*16`, 元素数量减少了 12 倍.

FLUX 中用于文字编码的有两个模型:
- clip, 一个参数在 125M 级别的小模型, 将提示词编码为 768 维向量, 作为全局条件指导生图的整体风格.
- t5, 一个参数在 4.7B 的大模型(仅 encoder 部分), 将提示词编码为 (seq_len, 4096) 的序列 (seq_len ≤ 512), 提供逐词的细粒度语义信息.

这是通过 `Claude Code` 从 `diffusers` 的代码中去除其他因素后抽象出来的推理过程.
可以看到, Transformer 预测当前 latent 中的噪声,
scheduler 根据预测的噪声和当前时间步, 计算去噪后的 latent.
```
  核心流程图

  prompt ──→ [CLIP] ──→ pooled_embeds (768,)
         ──→ [T5]   ──→ prompt_embeds (512, 4096)
                              ↓
  noise ──→ [Transformer × N steps] ──→ latents ──→ [VAE Decode] ──→ image

  伪代码

  def __call__(prompt, height=1024, width=1024, num_inference_steps=28):

      # 1. 文本编码
      pooled_embeds = CLIP(prompt)           # (768,)
      prompt_embeds = T5(prompt)             # (512, 4096)
      text_ids = zeros(512, 3)               # 文本位置 ID，三维分别为 (timestamp, height, width)，文本无空间位置故全零

      # 2. 准备潜变量
      latents = randn(1, 16, 128, 128)       # 随机噪声
      latents = pack(latents)                # → (1, 4096, 64)
      img_ids = prepare_image_ids(64, 64)    # 图像位置 ID，pack 后 latent 的高宽为 128/2=64

      # 3. 准备时间步
      timesteps = linspace(1.0, 0.036, 28)   # 从 1 到 ~0

      # 4. 去噪循环
      for t in timesteps:
          noise_pred = Transformer(
              hidden_states=latents,          # 当前潜变量
              timestep=t,                     # 当前时间步
              pooled_projections=pooled_embeds,    # CLIP 全局条件
              encoder_hidden_states=prompt_embeds,  # T5 序列条件
              txt_ids=text_ids,               # 文本位置
              img_ids=img_ids,                # 图像位置
          )

          latents = scheduler.step(noise_pred, t, latents)

      # 5. 解码
      latents = unpack(latents)              # (1, 4096, 64) → (1, 16, 128, 128)
      latents = latents / 0.3611 + 0.1159    # scaling + shift
      image = VAE.decode(latents)            # → (1, 3, 1024, 1024)

      return image
```

### Transformer
时至今日, `Attention Is All You Need` 这篇论文的知名度早已破圈,
其中提出的 Transformer 模型成为了现代 AI 的基石.
FLUX 中使用的 Transformer 核心机制依然是 QKV 的注意力机制.

Transformer 的核心是注意力，但作为一个入门的小白, 我们在 FLUX 中首先关注的是一些基础的概念.
这是 `Claude` 抽象的推理流程:
```python
  def forward(
      hidden_states,          # 图像 latents (1, 4096, 64)
      encoder_hidden_states,  # T5 嵌入 (1, 512, 4096)
      pooled_projections,     # CLIP 嵌入 (1, 768)
      timestep,               # 时间步
      img_ids,                # 图像位置 (4096, 3)
      txt_ids,                # 文本位置 (512, 3)
      guidance,               # guidance scale
  ):
      # 1. 嵌入
      hidden_states = x_embedder(hidden_states)              # (1, 4096, 3072)
      encoder_hidden_states = context_embedder(encoder_hidden_states)  # (1, 512, 3072)
      temb = time_text_embed(timestep, guidance, pooled_projections)   # 条件嵌入

      # 2. 位置编码
      ids = concat(txt_ids, img_ids)          # (4608, 3)
      rotary_emb = pos_embed(ids)             # RoPE

      # 3. 双流块 × 19
      for block in transformer_blocks:
          encoder_hidden_states, hidden_states = block(
              hidden_states,
              encoder_hidden_states,
              temb,
              rotary_emb,
          )

      # 4. 合并序列，进入单流块 × 38
      hidden_states = concat(encoder_hidden_states, hidden_states)  # (1, 4608, 3072)
      for block in single_transformer_blocks:
          hidden_states = block(hidden_states, temb, rotary_emb)

      # 5. 仅取图像部分
      hidden_states = hidden_states[:, txt_ids.shape[0]:]  # (1, 4096, 3072)

      # 6. 输出
      hidden_states = norm_out(hidden_states, temb)
      output = proj_out(hidden_states)        # (1, 4096, 64)

      return output
```

`x_embedder` 和 `context_embedder` 是 `torch.Linear`, 对应公式 `y = xW + b`,
分别将输入的 latents 和 t5's embed 投影到 3072 维, 即 `num_attention_heads(24) * attention_head_dim(128)`.
同样的, 在最后, 我们需要将输出通过 `proj_out` 从 3072 投影回 64.

`norm_out` 是 `AdaLayerNormContinuous`, 一个自适应归一化层.
归一化(LayerNorm)将向量调整为均值 0, 方差 1 的分布, 有助于训练稳定性.
这儿的"自适应"是指 scale 和 shift 参数由时间步(temb)动态生成, 而非固定的可学习参数.

`norm_out` 的具体计算为: `scale, shift = Linear(SiLU(temp)); output = LayerNorm(hidden_states) × (1 + scale) + shift`.
其中的新概念 SiLU, 是一种非线性的激活函数, 目的是为了让模型有非线性变换的能力.

RoPE, 旋转位置编码, 是 Transformer 中的一个重要概念.
其目的是为了让模型知道每个 token 的位置信息, 公式推导复杂, 但相比 attention 本身的计算量可以忽略.

除去上述, 剩下的就是两个 Attention Blocks, Transformer 的核心.

### Attention
我们以 `FluxSingleTransformerBlock` 为例来学习 Attention, 其抽象的推理流程如下:
```python
  代码定义 (第 356-406 行)

  class FluxSingleTransformerBlock(nn.Module):
      def __init__(self, dim, num_attention_heads, attention_head_dim, mlp_ratio=4.0):
          self.norm = AdaLayerNormZeroSingle(dim)
          self.proj_mlp = nn.Linear(dim, mlp_hidden_dim)  # 3072 → 12288
          self.act_mlp = nn.GELU()
          self.proj_out = nn.Linear(dim + mlp_hidden_dim, dim)  # 15360 → 3072
          self.attn = FluxAttention(...)

  简化流程（输入是已拼接的文本+图像序列）

  def forward(hidden_states, temb, rotary_emb):
      # hidden_states: (1, 4608, 3072)，已拼接的文本和图像

      residual = hidden_states

      # 1. 自适应归一化
      x_norm, gate = norm(hidden_states, temb)

      # 2. 并行计算 Attention 和 MLP
      attn_out = Attention(x_norm, rotary_emb)     # (1, 4608, 3072)
      mlp_out = GELU(Linear(x_norm))               # (1, 4608, 12288)

      # 3. 拼接 + 投影
      combined = concat(attn_out, mlp_out, dim=-1)  # (1, 4608, 15360)
      out = Linear(combined)                         # (1, 4608, 3072)

      # 4. 门控 + 残差
      hidden_states = residual + gate * out

      return hidden_states  # (1, 4608, 3072)
```
两个新的概念, 一个是 MLP, 多层感知机(Multi-Layer Perceptron), 在 Transformer 中我们也称呼其为 FFN, Feed-Forward Network.
其典型结构是 `Linear -> GELU -> Linear`, 在上述流程中最后一个 Linear 发生在 `concat(attn_out, mlp_out, dim=-1)` 之后.

另一个是 gate, 来自 `AdaLayerNormZeroSingle` 的返回, 这一层做了两件事.
一方面是归一化层, `norm(x) * scale + shift`.
另一方面是通过 gate, 控制更新的强度, 即 `new = old + gate * new`.
这两者同时都受到时间步 temb 的控制.

在使用经典公式 `softmax(Q × K^T / √d) × V` 计算注意力前, QKV 要经过一次投影, QK 要经过一次归一化.
```python
  def forward(hidden_states, rotary_emb):
      # hidden_states: (1, 4608, 3072)，拼接的文本+图像序列

      # 1. Q, K, V 投影
      Q = to_q(hidden_states)   # (1, 4608, 3072)
      K = to_k(hidden_states)
      V = to_v(hidden_states)

      # 2. 分成多头
      Q = Q.reshape(1, 4608, 24, 128)
      K = K.reshape(1, 4608, 24, 128)
      V = V.reshape(1, 4608, 24, 128)

      # 3. 归一化
      Q = norm_q(Q)
      K = norm_k(K)

      # 4. 旋转位置编码
      Q = Q * cos + rotate(Q) * sin
      K = K * cos + rotate(K) * sin

      # 5. Attention
      attn = softmax(Q @ K^T / √128) @ V

      # 6. 合并多头 + 输出
      out = attn.reshape(1, 4608, 3072)

      return out
```

### 小结

我们以 FLUX 为例，自顶向下学习 了文生图的核心流程：
- **Pipeline 层**：文本编码（CLIP + T5）→ 去噪循环（Transformer + Scheduler）→ 图像解码（VAE）
- **Transformer 层**：嵌入投影 → 位置编码 → 双流块（文本/图像分离）→ 单流块（文本/图像融合）→ 输出投影
- **Attention 层**：QKV 投影 → 多头拆分 → QK 归一化 → RoPE → Attention 计算

`softmax(Q @ K^T / √d) @ V`, 这个公式是 Transformer 计算量的主要来源，后续我们会结合 flash-attention 的优化来学习。
