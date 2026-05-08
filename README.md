# IntelliWhats 🤖

Assistente inteligente para WhatsApp que entende o que você envia - sejam imagens, áudios ou figurinhas - e responde automaticamente com informações úteis.

## 💡 O que ele faz?

O IntelliWhats é um bot que adiciona inteligência artificial ao seu WhatsApp:

- **📸 Descreve imagens**: Envie qualquer foto e receba uma descrição detalhada do que está nela
- **🎭 Explica figurinhas**: Não entendeu uma figurinha? O bot explica o conteúdo dela
- **🎤 Transcreve áudios**: Envie um áudio e receba o texto do que foi falado, já formatado e limpo

Perfeito para acessibilidade, produtividade ou simplesmente para facilitar sua vida no WhatsApp!

## 🚀 Como usar?

### Passo 1: Configurar suas chaves de API

Você vai precisar criar contas (gratuitas) em dois serviços de IA:

**Grok (da X.AI)**
- Acesse: https://console.x.ai/
- Crie sua conta e gere uma chave de API
- Guarde essa chave, você vai precisar dela

**Gemini (do Google)**
- Acesse: https://aistudio.google.com/apikey
- Entre com sua conta Google
- Clique em "Create API Key" e copie a chave

### Passo 2: Criar arquivo de configuração

Crie um arquivo chamado `.env` na pasta do projeto com o seguinte conteúdo:

```env
# Suas chaves de API
GROK_API_KEY=sua-chave-do-grok-aqui
GEMINI_API_KEY=sua-chave-do-gemini-aqui

# Seu número de WhatsApp (com código do país, sem espaços)
PHONE_NUMBER=5511999999999
```

### Passo 3: Iniciar o bot

**No Windows:**
```bash
docker-compose up -d
docker-compose logs -f
```

**No primeiro uso:**
- O bot vai mostrar um código de pareamento no terminal
- Abra o WhatsApp no celular → Aparelhos conectados → Conectar um aparelho
- Escolha "Conectar com número de telefone"
- Digite o código que apareceu no terminal
- Pronto! O bot está conectado

## 🎯 Como funciona na prática?

Uma vez conectado, é só usar normalmente:

1. **Para imagens**: Envie qualquer foto para o bot e ele responderá descrevendo o que vê
2. **Para áudios**: Envie um áudio e receba a transcrição em texto
3. **Para figurinhas**: Envie uma figurinha e ele explicará o conteúdo

Simples assim! Sem comandos complicados.

### Comandos disponíveis

- **!ping** - Verifica se o bot está ativo (ele responde "pong")
- **!api** - Mostra qual IA está sendo usada para processar imagens e figurinhas
- **!api 1** - Muda para usar Grok (padrão)
- **!api 2** - Muda para usar Gemini

> 💡 **Dica:** Você pode escolher qual IA prefere usar para processar suas imagens! Teste ambas e veja qual funciona melhor para você.

## 🔧 Comandos úteis

**Ver o que está acontecendo:**
```bash
docker-compose logs -f
```

**Parar o bot:**
```bash
docker-compose down
```

**Reiniciar o bot:**
```bash
docker-compose restart
```

**Desconectar e reconectar (gerar novo código):**
```bash
docker-compose down
# Apague a pasta 'data'
docker-compose up -d
docker-compose logs -f
```

## ❓ Perguntas frequentes

**O bot não está respondendo?**
- Verifique se o container está rodando: `docker-compose ps`
- Veja os logs para identificar erros: `docker-compose logs -f`
- Certifique-se de que suas chaves de API estão corretas no arquivo `.env`

**Preciso pagar para usar?**
- O bot em si é gratuito
- As APIs (Grok e Gemini) têm planos gratuitos, mas podem ter limites de uso
- Consulte os sites das APIs para mais informações sobre limites

**Posso usar em grupos?**
- Sim! Adicione o número do bot em qualquer grupo e ele responderá a todas as mensagens com mídia

**Os dados são privados?**
- As imagens e áudios são enviados para as APIs (Grok e Gemini) para processamento
- Nenhum dado é armazenado permanentemente pelo bot
- Consulte as políticas de privacidade da X.AI e Google para mais informações

## 🛠️ Requisitos técnicos

Para rodar o projeto, você precisa ter instalado:
- Docker e Docker Compose
- Conexão com internet (o bot precisa acessar as APIs)
