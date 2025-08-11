![fuffa mascot](_img/ffuf_run_logo_600.png)
# 🚀 fuffa - FFUF Using Fantastic Formats And colors

A fast web fuzzer written in Go, forked from ffuf with additional features and Italian localization.

## ⚡ Quick Start

```bash
# Basic directory fuzzing
fuffa -w wordlist.txt -u https://example.org/FUZZ

# Italian help
fuffa -aiuto

# With filtering and colors
fuffa -w wordlist.txt -u https://example.org/FUZZ -mc all -fs 42 -c -v
```

## ✨ What's New in fuffa
- 🇮🇹 **Italian localization** (`-aiuto` flag)
- 🎨 **Enhanced output formatting**
- 🔧 **Improved user experience**
- 📊 **Better progress indicators**


## 📦 Installation

### Option 1: Download Binary (Recommended)
```bash
# Download from releases (when available)
wget https://github.com/Mascol9/fuffa/releases/latest/download/fuffa_linux_amd64.tar.gz
tar -xzf fuffa_linux_amd64.tar.gz
chmod +x fuffa
./fuffa -V
```

### Option 2: Build from Source
```bash
# Clone the repository
git clone https://github.com/Mascol9/fuffa
cd fuffa

# Build
go build -ldflags "-s -w" -o fuffa .

# Run
./fuffa -V
```

**Requirements**: Go 1.17 or greater

## 🎯 Usage Examples

### 🔍 Directory Discovery
```bash
fuffa -w /path/to/wordlist -u https://target/FUZZ
```

### 🌐 Virtual Host Discovery
```bash
fuffa -w vhost-wordlist.txt -u https://target -H "Host: FUZZ" -fs 4242
```

### 🔍 Parameter Fuzzing
```bash
# GET parameters
fuffa -w params.txt -u https://target/script.php?FUZZ=test_value -fs 4242

# POST data
fuffa -w passwords.txt -X POST -d "username=admin&password=FUZZ" -u https://target/login.php -fc 401
```

### 🇮🇹 Italian Help
```bash
fuffa -aiuto  # Show help in Italian
```

---

## 📚 Documentation

### Configuration Files
fuffa supports configuration files at `$XDG_CONFIG_HOME/ffuf/ffufrc`. See [ffufrc.example](ffufrc.example) for reference.

### Advanced Usage
For complete documentation with all flags and options:
```bash
fuffa -h      # English help
fuffa -aiuto  # Italian help  
```

## 🤝 Contributing
Based on [ffuf](https://github.com/ffuf/ffuf) by [@joohoi](https://github.com/joohoi) and contributors.

## 📝 TODO
See [TODO.md](TODO.md) for planned improvements and features.
