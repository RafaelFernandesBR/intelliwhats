# WhatsApp Go Bridge Bot

Bot WhatsApp escrito em Go que processa automaticamente imagens, figurinhas e áudios via IA.

## 📁 Estrutura do Projeto

```
whatsappGo/
├── .env                     # ⚠️ Configurações sensíveis (não commitado)
├── .env.example             # Template para configuração
├── .gitignore               # Proteção de arquivos sensíveis
├── docker-compose.yml       # Configuração Docker Compose
├── docker-manage.bat        # Script helper Windows
├── docker-manage.sh         # Script helper Linux/Mac
├── data/                    # Volume Docker (sessão persistente)
└── whatsapp-bridge/         # Código fonte Go
    ├── *.go                # Código modularizado
    ├── Dockerfile          # Imagem Docker
    ├── .dockerignore       # Arquivos ignorados no build
    ├── README.md           # Documentação de uso
    ├── ARCHITECTURE.md     # Arquitetura do código
    ├── DOCKER.md           # Guia completo Docker
    └── SECURITY.md         # Guia de segurança
```

## 🚀 Início Rápido

### Opção 1: Docker (Recomendado)

**Pré-requisitos:**
- Docker e Docker Compose instalados

**Passos:**

```bash
# 1. Configurar .env
cp .env.example .env
nano .env  # Edite com suas API keys

# 2. Usar script helper
# Windows:
docker-manage.bat start

# Linux/Mac:
chmod +x docker-manage.sh
./docker-manage.sh start

# 3. Ver logs e código de pareamento
docker-manage.bat logs       # Windows
./docker-manage.sh logs      # Linux/Mac
```

**Comandos disponíveis:**
```bash
build      # Build da imagem Docker
start      # Iniciar o bot
stop       # Parar o bot
restart    # Reiniciar o bot
logs       # Ver logs em tempo real
status     # Ver status do container
shell      # Abrir shell no container
reset      # Reset completo (deleta sessão)
backup     # Backup da sessão
restore    # Restaurar backup
clean      # Limpar tudo
```

### Opção 2: Executável Local

**Pré-requisitos:**
- Go 1.25+ instalado
- GCC (para CGo/SQLite)

**Passos:**

```bash
cd whatsapp-bridge

# 1. Compilar
go build -o whatsapp-bot.exe .

# 2. Executar
# Windows:
run.bat

# Linux/Mac:
chmod +x run.sh
./run.sh
```

## ⚙️ Configuração

### Variáveis de Ambiente (.env)

**⚠️ Obrigatórias:**

```env
# API Keys (sempre obrigatórias)
GROK_API_KEY=xai-xxxxxxxxxxxxxxx        # Obter em: https://console.x.ai/
GEMINI_API_KEY=AIzaSyxxxxxxxxxxxxxxx    # Obter em: https://aistudio.google.com/apikey

# WhatsApp (apenas primeiro login)
PHONE_NUMBER=5511999999999              # Seu número com código do país
```

**📝 Opcionais:**

```env
OWNER_JID=5511999999999@s.whatsapp.net  # JID do proprietário
TZ=America/Sao_Paulo                    # Fuso horário
DB_PATH=/data/whatsmeow.db              # Caminho do banco (Docker)
```

### Obter API Keys

**Grok API (X.AI):**
1. Acesse: https://console.x.ai/
2. Crie uma conta
3. Vá em "API Keys"
4. Crie uma nova chave
5. Copie para `GROK_API_KEY` no `.env`

**Gemini API (Google):**
1. Acesse: https://aistudio.google.com/apikey
2. Faça login com Google
3. Clique em "Create API Key"
4. Copie para `GEMINI_API_KEY` no `.env`

## 🤖 Funcionalidades

### 🖼️ Descrição de Imagens
- Envia uma imagem → Recebe descrição detalhada em português
- Usa **Grok API** (modelo `grok-4-0709`)
- Ideal para usuários com deficiência visual

### � Descrição de Figurinhas (Stickers)
- Envia uma figurinha → Recebe descrição do conteúdo
- Usa **Grok API** (mesma tecnologia das imagens)
- Funciona com qualquer tipo de sticker

### �🎤 Transcrição de Áudios
- Envia um áudio → Recebe transcrição limpa em português
- Usa **Gemini API** (modelo `gemini-2.5-flash`)
- Remove hesitações e vícios de linguagem

### 💬 Comandos
- `!ping` → Responde "pong" (teste de conectividade)

## 📖 Documentação

- **[README.md](whatsapp-bridge/README.md)** - Guia de uso e configuração
- **[ARCHITECTURE.md](whatsapp-bridge/ARCHITECTURE.md)** - Arquitetura do código
- **[DOCKER.md](whatsapp-bridge/DOCKER.md)** - Guia completo Docker
- **[SECURITY.md](whatsapp-bridge/SECURITY.md)** - Segurança de API keys
- **[REFACTORING.md](whatsapp-bridge/REFACTORING.md)** - Documentação da refatoração
- **[ENV_MIGRATION.md](whatsapp-bridge/ENV_MIGRATION.md)** - Migração para variáveis de ambiente

## 🔐 Segurança

### ⚠️ IMPORTANTE

1. **Nunca commite .env para Git**
   - Já está no `.gitignore`
   - Contém credenciais sensíveis

2. **Use .env.example como referência**
   - Template sem credenciais reais
   - Safe para commit

3. **Rotacione chaves periodicamente**
   - Especialmente se suspeitar de exposição

4. **Docker: .dockerignore protege .env**
   - Arquivo não é copiado para imagem
   - Variáveis vêm do docker-compose

### Checklist de Segurança

- [ ] `.env` está no `.gitignore` ✅
- [ ] `.env.example` não tem chaves reais ✅
- [ ] API keys em variáveis de ambiente ✅
- [ ] `.dockerignore` protege arquivos sensíveis ✅
- [ ] Scripts validam presença de chaves ✅

## 🐛 Troubleshooting

### Erro: "GROK_API_KEY não definida"

**Causa:** Arquivo `.env` não foi criado ou está vazio.

**Solução:**
```bash
# Copiar exemplo
cp .env.example .env

# Editar e adicionar suas chaves
nano .env

# Reiniciar
docker-manage.bat restart  # ou ./docker-manage.sh restart
```

### Bot não conecta no primeiro uso

**Causa:** `PHONE_NUMBER` não configurado ou código de pareamento expirado.

**Solução:**
```bash
# 1. Verificar .env
cat .env | grep PHONE_NUMBER

# 2. Ver logs para código de pareamento
docker-manage.bat logs  # ou ./docker-manage.sh logs

# 3. Digitar código rapidamente no WhatsApp:
#    Configurações → Aparelhos conectados → Conectar com número
```

### Reset da sessão (novo pareamento)

```bash
# Docker
docker-manage.bat reset  # ou ./docker-manage.sh reset

# Local
rm -rf whatsapp-bridge/login/
```

## 🔄 Comparação: Docker vs Local

| Aspecto | Docker | Local |
|---------|--------|-------|
| **Configuração** | Mais fácil | Mais complexa |
| **Dependências** | Isoladas | Precisa instalar |
| **Portabilidade** | ✅ Alta | ⚠️ Depende do SO |
| **Atualização** | `docker-compose build` | `go build` |
| **Logs** | `docker-compose logs` | Terminal direto |
| **Backup** | Script helper | Manual |
| **Produção** | ✅ Recomendado | ⚠️ Ok para dev |

## 📦 Dependências Go

```go
go.mau.fi/whatsmeow          // Cliente WhatsApp
github.com/mattn/go-sqlite3  // Banco de dados SQLite
google.golang.org/protobuf   // Protocol Buffers
```

## 🏗️ Arquitetura

### Modularização (7 arquivos Go)

```
whatsapp-bridge/
├── main.go        # Entry point e orquestração
├── config.go      # Configurações e constantes
├── auth.go        # Autenticação WhatsApp
├── handlers.go    # Handlers de eventos
├── grok.go        # Integração Grok API
├── gemini.go      # Integração Gemini API
└── utils.go       # Funções utilitárias
```

**Benefícios:**
- ✅ Separação de responsabilidades
- ✅ Fácil manutenção
- ✅ Testabilidade melhorada
- ✅ Código limpo e legível

## 🚢 Deploy em Produção

### Docker Compose (Recomendado)

```bash
# 1. Servidor
git clone <repo> /opt/whatsapp-bot
cd /opt/whatsapp-bot

# 2. Configurar .env
cp .env.example .env
nano .env

# 3. Deploy
docker-compose up -d

# 4. Monitorar
docker-compose logs -f
```

### Backup Automatizado

```bash
# Backup diário via cron
0 3 * * * /opt/whatsapp-bot/docker-manage.sh backup
```

## 📊 Status do Projeto

- ✅ Autenticação por código (sem QR)
- ✅ Processamento de imagens (Grok)
- ✅ Processamento de figurinhas/stickers (Grok)
- ✅ Transcrição de áudios (Gemini)
- ✅ Comando !ping
- ✅ API keys em variáveis de ambiente
- ✅ Código refatorado e modular
- ✅ Documentação completa
- ✅ Docker pronto para produção
- ✅ Scripts helper (Windows/Linux/Mac)
- ✅ Sistema de backup

## 📝 Licença

Projeto educacional demonstrando integração WhatsApp + IA.

## 🤝 Suporte

Para problemas ou dúvidas:
1. Consulte [DOCKER.md](whatsapp-bridge/DOCKER.md) para troubleshooting Docker
2. Consulte [SECURITY.md](whatsapp-bridge/SECURITY.md) para questões de segurança
3. Veja [ARCHITECTURE.md](whatsapp-bridge/ARCHITECTURE.md) para entender o código

---

**Bot pronto para uso!** 🚀🤖
