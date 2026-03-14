# 🚀 AYWS Compute Service

**AYWS altyapısı** için özel olarak geliştirilmiş, **Proxmox** ve **Docker** gücünü tek bir merkezde birleştiren dinamik bir hesaplama (compute) ve orkestrasyon servisidir. 

Bu sistem, donanım kaynaklarını sanallaştırma ve konteynerleştirme teknolojileri üzerinden tamamen otomatik bir şekilde yöneterek, uygulamalarınız için en hızlı, en güvenilir ve en verimli kaynak tahsisini yapmanızı sağlar. Manuel müdahaleleri ortadan kaldırır ve altyapınızı kod üzerinden (Infrastructure as Code) yönetilebilir hale getirir. 🏗️🌐

---

## ⚡ Sistem Neler Yapabiliyor?

Bu servis, arka plandaki karmaşık altyapı süreçlerini soyutlar ve otomatize ederek aşağıdaki tüm işlemleri doğrudan **RESTful API** üzerinden gerçekleştirmenize olanak tanır:

### 🖥️ Proxmox Node Yönetimi
Proxmox hipervizörü ile pürüzsüz bir entegrasyon sunar.
* **Hızlı Kurulum:** Proxmox API'si ile doğrudan ve güvenli bir şekilde haberleşerek, saniyeler içinde yeni **Sanal Makineler (VM)** ve **LXC konteynerler** oluşturur.
* **Tam Kontrol:** Oluşturulan makineleri başlatma, durdurma, yeniden başlatma (reboot) veya silme işlemlerini tek bir API çağrısıyla halleder.
* **Şablonlama (Templating):** Hazır imajlar ve şablonlar üzerinden hızlı klonlama yaparak altyapı hazırlık süresini minimuma indirir.

### 🐳 Docker Konteyner Orkestrasyonu
Uygulamalarınızın izolasyonu ve hızlı dağıtımı için Docker daemon ile birebir çalışır.
* **Anında Dağıtım (Deployment):** Hedef sunucularda Docker ile etkileşime geçerek uygulamaları izole konteynerler halinde anında ayağa kaldırır.
* **Yaşam Döngüsü (Lifecycle) Yönetimi:** Konteynerlerin başlatılması, durdurulması, loglarının okunması ve imha edilmesi süreçlerini uçtan uca yönetir.
* **Ağ ve Volume Yönetimi:** Konteynerler arası ağ izolasyonunu ve kalıcı veri (persistent storage) için volume bağlamalarını otomatik yapılandırır.

### ⚙️ Dinamik Kaynak Tahsisi
Donanım israfını önler ve performansı optimize eder.
* **Esnek Ölçeklendirme:** Dışarıdan gelen isteklere ve anlık ihtiyaçlara göre **CPU çekirdek sayısı, RAM miktarı ve Disk kapasitelerini** dinamik olarak ayarlar.
* **Anında Güncelleme:** Çalışan sistemler üzerinde (desteklenen senaryolarda) kaynak güncellemeleri yaparak kesintisiz dikey ölçeklendirme sağlar.
* **Akıllı Dağıtım:** Kaynakları, host sunucular (node'lar) üzerinde en verimli şekilde dağıtarak darboğaz (bottleneck) oluşmasını engeller.

### 📊 Durum ve Metrik İzleme
Kör uçuşu yapmanızı engeller, sistemin anlık röntgenini çeker.
* **Health Check (Sağlık Durumu):** Çalışan sanal makinelerin (VM/LXC) ve Docker konteynerlerinin anlık olarak ayakta olup olmadığını (Up/Down) takip eder.
* **Kaynak Tüketimi:** CPU kullanımı, bellek (RAM) doluluğu ve disk I/O gibi kritik performans metriklerini anlık olarak izler ve raporlar.
* **Proaktif Yönetim:** API üzerinden çekilebilen bu metrikler sayesinde, sistemin aşırı yüklenmesi durumunda otomatik aksiyonlar alınmasına zemin hazırlar.

### 🔌 Merkezi Altyapı Kontrolü
Karmaşık arayüzlere veda edin; her şey tek bir merkezden yönetilir.
* **RESTful API Uç Noktaları:** Tüm sunucu, VM ve konteyner yönetimi standart HTTP istekleri (GET, POST, PUT, DELETE) ile yapılabilir hale gelir.
* **Kolay Entegrasyon:** Diğer AYWS servislerinin, frontend arayüzlerinin veya **CI/CD pipeline'larının (GitHub Actions, GitLab CI vb.)** kolayca tetikleyebileceği bir yapı sunar.
* **Bütünleşik Orkestrasyon:** Geliştiricilerin Proxmox arayüzüne veya sunucu terminallerine girmesine gerek kalmadan, kendi yazdıkları kodlar üzerinden altyapı talep etmelerini sağlar.

---

## 🛠️ Temel Teknolojiler

Altyapının kalbinde yer alan sistemler:
* 🛡️ **Proxmox VE:** Donanım sanallaştırma ve LXC yönetimi.
* 📦 **Docker:** Uygulama seviyesinde konteynerizasyon.
* 🌐 **REST API:** Node'lar ile iletişimi sağlayan haberleşme katmanı.
* 🔐 **JSON/HTTP:** Standartlaştırılmış ve hızlı veri transferi.
