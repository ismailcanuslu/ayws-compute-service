#🚀 AYWS Compute Service
AYWS altyapısı için geliştirilmiş, Proxmox ve Docker gücünü tek bir merkezde birleştiren dinamik bir hesaplama (compute) ve orkestrasyon servisidir. Sistem, donanım kaynaklarını sanallaştırma ve konteynerleştirme teknolojileri üzerinden yöneterek, uygulamalarınız için hızlı ve güvenilir kaynak tahsisi yapmanızı sağlar.

##⚡ Sistem Neler Yapabiliyor?
Bu servis, altyapı süreçlerini otomatize ederek aşağıdaki işlemleri doğrudan API üzerinden gerçekleştirir:

🖥️ Proxmox Node Yönetimi: Proxmox API'si ile doğrudan haberleşerek saniyeler içinde yeni Sanal Makineler (VM) ve LXC konteynerler oluşturur, başlatır veya durdurur.

🐳 Docker Konteyner Orkestrasyonu: Hedef sunucularda Docker daemon ile etkileşime geçerek uygulamaları izole konteynerler halinde anında ayağa kaldırır ve yaşam döngülerini (lifecycle) yönetir.

⚙️ Dinamik Kaynak Tahsisi: İhtiyaca göre CPU, RAM ve Disk kapasitelerini dışarıdan gelen isteklere göre ayarlar, günceller ve sunuculara dağıtır.

📊 Durum ve Metrik İzleme: Çalışan sanal makinelerin ve Docker konteynerlerinin anlık "health" (sağlık) durumlarını ve kaynak tüketim metriklerini takip eder.

🔌 Merkezi Altyapı Kontrolü: Tüm sunucu ve konteyner yönetimini karmaşık arayüzler yerine, diğer AYWS servislerinin veya CI/CD süreçlerinin kolayca tetikleyebileceği RESTful uç noktaları (endpoint) üzerinden sunar.
