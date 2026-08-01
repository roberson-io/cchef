package ops

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Oracle-derived vectors (CyberChef-server) — Parse X.509 certificate has no
// upstream fixture file. PEMs are fixed self-signed test certificates.

const (
	certRSAPEM = `-----BEGIN CERTIFICATE-----
MIIEIjCCAwqgAwIBAgIUOPZEMuWcRV484CMIJyF0p8ZQsvYwDQYJKoZIhvcNAQEL
BQAwcjELMAkGA1UEBhMCQ0gxDzANBgNVBAgMBlp1cmljaDEPMA0GA1UEBwwGWnVy
aWNoMRMwEQYDVQQKDApFeGFtcGxlIFJFMRYwFAYDVQQLDA1JVCBEZXBhcnRtZW50
MRQwEgYDVQQDDAtleGFtcGxlLmNvbTAeFw0yNjA3MjAwMTQyMjNaFw0zNjA3MTcw
MTQyMjNaMHIxCzAJBgNVBAYTAkNIMQ8wDQYDVQQIDAZadXJpY2gxDzANBgNVBAcM
Blp1cmljaDETMBEGA1UECgwKRXhhbXBsZSBSRTEWMBQGA1UECwwNSVQgRGVwYXJ0
bWVudDEUMBIGA1UEAwwLZXhhbXBsZS5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IB
DwAwggEKAoIBAQC40QhbQkmRtTj6r/KHBXr1RTI5VVvBiwTXHPP2vJgCIDsUbFR+
N44tevqIeLYRynwsSkbMtCGwCylBDZq5TROtuMHCX/sIDwITIHYf/m4gqR8TPe4H
kTvHh9CNta2eOqRZgPbbyn62rArlg0wlyO5JbAoAFvCDm0xCu2VF8pc4wYgpgnRg
I8ari26FXGlG/kKfmncrkosXb4xZL07XBBUuLgC/xgKxi7unlG4As2MZG6givvLd
R/jXxLW2kkmF7r5WFbIbV9WGJVE3e5IY03oH81X50XCJY7zcuSmW6PYLqnWKfH7m
8QL68zCQYh3KsHDi8EEqagbiP4OA/uaSvIqvAgMBAAGjga8wgawwHQYDVR0OBBYE
FLeCQA5G8stxVK7hpk2TSUg0uU0CMB8GA1UdIwQYMBaAFLeCQA5G8stxVK7hpk2T
SUg0uU0CMC0GA1UdEQQmMCSCC2V4YW1wbGUuY29tgg93d3cuZXhhbXBsZS5jb22H
BH8AAAEwDAYDVR0TAQH/BAIwADAOBgNVHQ8BAf8EBAMCBaAwHQYDVR0lBBYwFAYI
KwYBBQUHAwEGCCsGAQUFBwMCMA0GCSqGSIb3DQEBCwUAA4IBAQB6MAyHAsg7Cc4N
CYdk5lpbvXtuTgeD3FBOZV4KeRpqstvajghWCGkSMQoHvqpdaJULwas8HR3jcKXs
Ws/yzz5V35ZPYzUfayX/Z6tmsXTpWIcE+aTNKQwROyCE48Menor+1B0b1JtsXZi8
Y9hfpqpjCS53kd0X2loa7y5BMRSM4iU6MJPc7oI2cYn8ug3SQNGKKChhoOWD2jpm
WqKT/eFgjPJq8cq/MjFjo0H865/vU0RewZo8oEWRZ2RFe9isn5RKfxHb4a/RataU
RfP/+sqiVXqILlXfM5nac6EUB9ixcjRq/1mlBhYFQcte4PL75n2xiGGHKW6WZ4d1
qP1/bVUB
-----END CERTIFICATE-----`
	certRSAOut = `Version:          3 (0x02)
Serial number:    325195407439739868690796149162879408866195452662 (0x38f64432e59c455e3ce02308272174a7c650b2f6)
Algorithm ID:     SHA256withRSA
Validity
  Not Before:     20/07/2026 01:42:23 (dd-mm-yyyy hh:mm:ss) (260720014223Z)
  Not After:      17/07/2036 01:42:23 (dd-mm-yyyy hh:mm:ss) (360717014223Z)
Issuer
  C  = CH
  ST = Zurich
  L  = Zurich
  O  = Example RE
  OU = IT Department
  CN = example.com
Subject
  C  = CH
  ST = Zurich
  L  = Zurich
  O  = Example RE
  OU = IT Department
  CN = example.com
Fingerprints
  MD5:            aba1ccd5f1325ae134a54910b4edcc58
  SHA1:           bcce6fa826847253c036a5e6e16f035b3ba7284d
  SHA256:         d6dc6c7b55b4b8b413756dafac676f2b9b3eaba20c3702227362548166d762b7
Public Key
  Algorithm:      RSA
  Length:         2048 bits
  Modulus:        b8:d1:08:5b:42:49:91:b5:38:fa:af:f2:87:05:7a:f5:
                  45:32:39:55:5b:c1:8b:04:d7:1c:f3:f6:bc:98:02:20:
                  3b:14:6c:54:7e:37:8e:2d:7a:fa:88:78:b6:11:ca:7c:
                  2c:4a:46:cc:b4:21:b0:0b:29:41:0d:9a:b9:4d:13:ad:
                  b8:c1:c2:5f:fb:08:0f:02:13:20:76:1f:fe:6e:20:a9:
                  1f:13:3d:ee:07:91:3b:c7:87:d0:8d:b5:ad:9e:3a:a4:
                  59:80:f6:db:ca:7e:b6:ac:0a:e5:83:4c:25:c8:ee:49:
                  6c:0a:00:16:f0:83:9b:4c:42:bb:65:45:f2:97:38:c1:
                  88:29:82:74:60:23:c6:ab:8b:6e:85:5c:69:46:fe:42:
                  9f:9a:77:2b:92:8b:17:6f:8c:59:2f:4e:d7:04:15:2e:
                  2e:00:bf:c6:02:b1:8b:bb:a7:94:6e:00:b3:63:19:1b:
                  a8:22:be:f2:dd:47:f8:d7:c4:b5:b6:92:49:85:ee:be:
                  56:15:b2:1b:57:d5:86:25:51:37:7b:92:18:d3:7a:07:
                  f3:55:f9:d1:70:89:63:bc:dc:b9:29:96:e8:f6:0b:aa:
                  75:8a:7c:7e:e6:f1:02:fa:f3:30:90:62:1d:ca:b0:70:
                  e2:f0:41:2a:6a:06:e2:3f:83:80:fe:e6:92:bc:8a:af
  Exponent:       65537 (0x10001)
Certificate Signature
  Algorithm:      SHA256withRSA
  Signature:      7a:30:0c:87:02:c8:3b:09:ce:0d:09:87:64:e6:5a:5b:
                  bd:7b:6e:4e:07:83:dc:50:4e:65:5e:0a:79:1a:6a:b2:
                  db:da:8e:08:56:08:69:12:31:0a:07:be:aa:5d:68:95:
                  0b:c1:ab:3c:1d:1d:e3:70:a5:ec:5a:cf:f2:cf:3e:55:
                  df:96:4f:63:35:1f:6b:25:ff:67:ab:66:b1:74:e9:58:
                  87:04:f9:a4:cd:29:0c:11:3b:20:84:e3:c3:1e:9e:8a:
                  fe:d4:1d:1b:d4:9b:6c:5d:98:bc:63:d8:5f:a6:aa:63:
                  09:2e:77:91:dd:17:da:5a:1a:ef:2e:41:31:14:8c:e2:
                  25:3a:30:93:dc:ee:82:36:71:89:fc:ba:0d:d2:40:d1:
                  8a:28:28:61:a0:e5:83:da:3a:66:5a:a2:93:fd:e1:60:
                  8c:f2:6a:f1:ca:bf:32:31:63:a3:41:fc:eb:9f:ef:53:
                  44:5e:c1:9a:3c:a0:45:91:67:64:45:7b:d8:ac:9f:94:
                  4a:7f:11:db:e1:af:d1:6a:d6:94:45:f3:ff:fa:ca:a2:
                  55:7a:88:2e:55:df:33:99:da:73:a1:14:07:d8:b1:72:
                  34:6a:ff:59:a5:06:16:05:41:cb:5e:e0:f2:fb:e6:7d:
                  b1:88:61:87:29:6e:96:67:87:75:a8:fd:7f:6d:55:01

Extensions
  subjectKeyIdentifier :
    b782400e46f2cb7154aee1a64d93494834b94d02
  authorityKeyIdentifier :
    kid=b782400e46f2cb7154aee1a64d93494834b94d02
  subjectAltName :
    dns: example.com
    dns: www.example.com
    ip: 127.0.0.1
  basicConstraints CRITICAL:
    {}
  keyUsage CRITICAL:
    digitalSignature,keyEncipherment
  extKeyUsage :
    serverAuth, clientAuth
`
	certECPEM = `-----BEGIN CERTIFICATE-----
MIIBoTCCAUigAwIBAgIUUbt4p+snZ9d9DBQWTjKyFmm0tOEwCgYIKoZIzj0EAwIw
GTEXMBUGA1UEAwwOZWMuZXhhbXBsZS5jb20wHhcNMjYwNzIwMDE0NzQ4WhcNMzYw
NzE3MDE0NzQ4WjAZMRcwFQYDVQQDDA5lYy5leGFtcGxlLmNvbTBZMBMGByqGSM49
AgEGCCqGSM49AwEHA0IABDKXHC0n6dzBiY2tDSmHaFfCitBJb65aJ11ibbHoYCbt
rs0EO6nhuzKZj+drnsdEWQ6Sf3Y5Jbcy9A2QZIfCWOCjbjBsMB0GA1UdDgQWBBQ2
pL2og0XzQ/4Uk6zExqzfHGRV5DAfBgNVHSMEGDAWgBQ2pL2og0XzQ/4Uk6zExqzf
HGRV5DAPBgNVHRMBAf8EBTADAQH/MBkGA1UdEQQSMBCCDmVjLmV4YW1wbGUuY29t
MAoGCCqGSM49BAMCA0cAMEQCIB8wxqQSDv4YxmWGlKsRoNPdD1XfE+J/QkZDJuOV
a+oJAiARwKHRcLXSLPBglXUuj2ITMZMkNOaNU2yt3yXeFVouwQ==
-----END CERTIFICATE-----`
	certECOut = `Version:          3 (0x02)
Serial number:    466609002402896499764555259965211308564262728929 (0x51bb78a7eb2767d77d0c14164e32b21669b4b4e1)
Algorithm ID:     SHA256withECDSA
Validity
  Not Before:     20/07/2026 01:47:48 (dd-mm-yyyy hh:mm:ss) (260720014748Z)
  Not After:      17/07/2036 01:47:48 (dd-mm-yyyy hh:mm:ss) (360717014748Z)
Issuer
  CN = ec.example.com
Subject
  CN = ec.example.com
Fingerprints
  MD5:            38c515b42874b226e7966a77a8823417
  SHA1:           fc6f07a25726d56bc17f208eb6d5ff94c05e513a
  SHA256:         4ab3d121b80b7794938e0506081c867f85b096feee098054c87c5e31ea7d1ac7
Public Key
  Algorithm:      EC
  Curve Name:     secp256r1
  Length:         256 bits
  pub:            04:32:97:1c:2d:27:e9:dc:c1:89:8d:ad:0d:29:87:68:
                  57:c2:8a:d0:49:6f:ae:5a:27:5d:62:6d:b1:e8:60:26:
                  ed:ae:cd:04:3b:a9:e1:bb:32:99:8f:e7:6b:9e:c7:44:
                  59:0e:92:7f:76:39:25:b7:32:f4:0d:90:64:87:c2:58:
                  e0
Certificate Signature
  Algorithm:      SHA256withECDSA
  r:              1f:30:c6:a4:12:0e:fe:18:c6:65:86:94:ab:11:a0:d3:
                  dd:0f:55:df:13:e2:7f:42:46:43:26:e3:95:6b:ea:09
  s:              7f:42:46:43:26:e3:95:6b:ea:09:02:20:11:c0:a1:d1:
                  70:b5:d2:2c:f0:60:95:75:2e:8f:62:13:31:93:24:34:
                  e6:8d:53:6c:ad:df:25:de:15:5a:2e:c1

Extensions
  subjectKeyIdentifier :
    36a4bda88345f343fe1493acc4c6acdf1c6455e4
  authorityKeyIdentifier :
    kid=36a4bda88345f343fe1493acc4c6acdf1c6455e4
  basicConstraints CRITICAL:
    cA=true
  subjectAltName :
    dns: ec.example.com
`
	certCAPEM = `-----BEGIN CERTIFICATE-----
MIIDuTCCAqGgAwIBAgIUBv9q6aAA/zcAMbPk+tQQrYLSUk0wDQYJKoZIhvcNAQEL
BQAwEjEQMA4GA1UEAwwHVGVzdCBDQTAeFw0yNjA3MjAwMTQ3NDhaFw0zNjA3MTcw
MTQ3NDhaMBIxEDAOBgNVBAMMB1Rlc3QgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IB
DwAwggEKAoIBAQDYqSlcIqnXIl8jHpl7hNo6rn6mxSgn9sP+KaCnyVK1CvxY/E1l
WyDTrRHk4UW+6hwqMSD1eeul4i76KSHG+QMYn72cNS1ixa/Vqi2nCWAgWochhN/Z
nbOFvFxu5a7g3PQdAnyJyz0GXX78bcR/K1WNwLRV61PrhQMPd13Ea/zr/RPAJgy2
i4AUO/vm7hzbWsoX7ic25F9Iofx7PP9Kkto8R3W7VONPHkGzLqD5O/4zKKXBHUsq
eJ4X10xLqe2NM+VOD5Ehx3IENd69W0KCG0XNvTdTeqqxMo3oQizUhT6l2C8zA0jj
4LqY0/tGz8RVsVUN1/EJG5eNGEttDnswrMjpAgMBAAGjggEFMIIBATAdBgNVHQ4E
FgQUr8A6QnxsvuykKQDGJWBVOEnivBcwHwYDVR0jBBgwFoAUr8A6QnxsvuykKQDG
JWBVOEnivBcwEgYDVR0TAQH/BAgwBgEB/wIBAzAOBgNVHQ8BAf8EBAMCAQYwLAYD
VR0fBCUwIzAhoB+gHYYbaHR0cDovL2V4YW1wbGUuY29tL3Jvb3QuY3JsMFoGCCsG
AQUFBwEBBE4wTDAjBggrBgEFBQcwAYYXaHR0cDovL29jc3AuZXhhbXBsZS5jb20w
JQYIKwYBBQUHMAKGGWh0dHA6Ly9leGFtcGxlLmNvbS9jYS5jcnQwEQYDVR0gBAow
CDAGBgQqAwQFMA0GCSqGSIb3DQEBCwUAA4IBAQCtIz28W2DbM3wLS6K/71tTRkMH
2Cu216MZibGqmXNqmwAkNhpQP54HowmwTiyNep6enoiHf3zL2oYmNG9AgWI5lw9b
mtQ3iwvhGudmZ/FJhLKOF0ufJP1kwllfhncDZ+7i87vXuLNEJvBkHQvGx/o+DjKf
AXGFgnST5m1YfgXTG6DNWz+/NC+V2rgZT73Oxtn07Z+CBEm1VkludqAgMoaDdbpk
BoemL8hEW5/syMh83Yqpa7CSONr1+X6qB6BVtYnpMqZi85mC8RlZ4PadFkrYGwgy
M2yNGAt6bMtQjLoXYNA+cnWmDoKP7aoZeLhF/AyEYlHPX/eriQiwT2Cjwxe4
-----END CERTIFICATE-----`
	certCAOut = `Version:          3 (0x02)
Serial number:    39949948051350260908941496543743108204027073101 (0x06ff6ae9a000ff370031b3e4fad410ad82d2524d)
Algorithm ID:     SHA256withRSA
Validity
  Not Before:     20/07/2026 01:47:48 (dd-mm-yyyy hh:mm:ss) (260720014748Z)
  Not After:      17/07/2036 01:47:48 (dd-mm-yyyy hh:mm:ss) (360717014748Z)
Issuer
  CN = Test CA
Subject
  CN = Test CA
Fingerprints
  MD5:            3e5b89d10fae1684c31a259bdcb2cffa
  SHA1:           7335b64c73f35b214b82ded1794b5ab3164077db
  SHA256:         fb0c875e6878c2c03c0310fd1cffcb0a8461847afdd84702ec9adac2e357aa31
Public Key
  Algorithm:      RSA
  Length:         2048 bits
  Modulus:        d8:a9:29:5c:22:a9:d7:22:5f:23:1e:99:7b:84:da:3a:
                  ae:7e:a6:c5:28:27:f6:c3:fe:29:a0:a7:c9:52:b5:0a:
                  fc:58:fc:4d:65:5b:20:d3:ad:11:e4:e1:45:be:ea:1c:
                  2a:31:20:f5:79:eb:a5:e2:2e:fa:29:21:c6:f9:03:18:
                  9f:bd:9c:35:2d:62:c5:af:d5:aa:2d:a7:09:60:20:5a:
                  87:21:84:df:d9:9d:b3:85:bc:5c:6e:e5:ae:e0:dc:f4:
                  1d:02:7c:89:cb:3d:06:5d:7e:fc:6d:c4:7f:2b:55:8d:
                  c0:b4:55:eb:53:eb:85:03:0f:77:5d:c4:6b:fc:eb:fd:
                  13:c0:26:0c:b6:8b:80:14:3b:fb:e6:ee:1c:db:5a:ca:
                  17:ee:27:36:e4:5f:48:a1:fc:7b:3c:ff:4a:92:da:3c:
                  47:75:bb:54:e3:4f:1e:41:b3:2e:a0:f9:3b:fe:33:28:
                  a5:c1:1d:4b:2a:78:9e:17:d7:4c:4b:a9:ed:8d:33:e5:
                  4e:0f:91:21:c7:72:04:35:de:bd:5b:42:82:1b:45:cd:
                  bd:37:53:7a:aa:b1:32:8d:e8:42:2c:d4:85:3e:a5:d8:
                  2f:33:03:48:e3:e0:ba:98:d3:fb:46:cf:c4:55:b1:55:
                  0d:d7:f1:09:1b:97:8d:18:4b:6d:0e:7b:30:ac:c8:e9
  Exponent:       65537 (0x10001)
Certificate Signature
  Algorithm:      SHA256withRSA
  Signature:      ad:23:3d:bc:5b:60:db:33:7c:0b:4b:a2:bf:ef:5b:53:
                  46:43:07:d8:2b:b6:d7:a3:19:89:b1:aa:99:73:6a:9b:
                  00:24:36:1a:50:3f:9e:07:a3:09:b0:4e:2c:8d:7a:9e:
                  9e:9e:88:87:7f:7c:cb:da:86:26:34:6f:40:81:62:39:
                  97:0f:5b:9a:d4:37:8b:0b:e1:1a:e7:66:67:f1:49:84:
                  b2:8e:17:4b:9f:24:fd:64:c2:59:5f:86:77:03:67:ee:
                  e2:f3:bb:d7:b8:b3:44:26:f0:64:1d:0b:c6:c7:fa:3e:
                  0e:32:9f:01:71:85:82:74:93:e6:6d:58:7e:05:d3:1b:
                  a0:cd:5b:3f:bf:34:2f:95:da:b8:19:4f:bd:ce:c6:d9:
                  f4:ed:9f:82:04:49:b5:56:49:6e:76:a0:20:32:86:83:
                  75:ba:64:06:87:a6:2f:c8:44:5b:9f:ec:c8:c8:7c:dd:
                  8a:a9:6b:b0:92:38:da:f5:f9:7e:aa:07:a0:55:b5:89:
                  e9:32:a6:62:f3:99:82:f1:19:59:e0:f6:9d:16:4a:d8:
                  1b:08:32:33:6c:8d:18:0b:7a:6c:cb:50:8c:ba:17:60:
                  d0:3e:72:75:a6:0e:82:8f:ed:aa:19:78:b8:45:fc:0c:
                  84:62:51:cf:5f:f7:ab:89:08:b0:4f:60:a3:c3:17:b8

Extensions
  subjectKeyIdentifier :
    afc03a427c6cbeeca42900c62560553849e2bc17
  authorityKeyIdentifier :
    kid=afc03a427c6cbeeca42900c62560553849e2bc17
  basicConstraints CRITICAL:
    cA=true, pathLen=3
  keyUsage CRITICAL:
    keyCertSign,cRLSign
  cRLDistributionPoints :
    http://example.com/root.crl
  authorityInfoAccess :
    ocsp: http://ocsp.example.com
    caissuer: http://example.com/ca.crt
  certificatePolicies :
    policy oid: 1.2.3.4.5
`
	certRichPEM = `-----BEGIN CERTIFICATE-----
MIIDzTCCArWgAwIBAgIUbpcTiMCV53sMZmioc3TsIgt5f7MwDQYJKoZIhvcNAQEL
BQAwKDELMAkGA1UEBhMCVVMxGTAXBgNVBAMMEHJpY2guZXhhbXBsZS5jb20wHhcN
MjYwNzIwMDE1MTM0WhcNMzYwNzE3MDE1MTM0WjAoMQswCQYDVQQGEwJVUzEZMBcG
A1UEAwwQcmljaC5leGFtcGxlLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
AQoCggEBAIWJDQ8Naym5xCVztJ0DS2MCtrUzdUmr1Wvg0Ynh7EzkunRhh++jGNAb
tjZ071zGjZ/U6nGQ5f1EDT/H9r3D9x6e/XkGkVRnQZ1RVGcRwWkdSijlfbxPnCVq
dE4Ns/JYGbVBm9vzZbn34tZ6iUb/TzyPi9D6S5hfPE9VZxW/uzKVUp5sW2M/qujx
u2ApTJdIacfkrA9ww7JDs6FnZSCbF7/jd/3b3SigoOkULMm5XzxSPVaesc7bGpGs
t3gFKgndm59EsuyZXQNdIoqVHXnOGV0YHXl5AwJljdu8Y9fdpjjFwXBvOKu4Wp+d
YDnFdLTjtU65mUhuwbwLmLQb5AEih6ECAwEAAaOB7jCB6zAdBgNVHQ4EFgQUdqB+
fnVae3o/dZYm4ZYS1DL62T0wHwYDVR0jBBgwFoAUdqB+fnVae3o/dZYm4ZYS1DL6
2T0wEgYDVR0TAQH/BAgwBgEB/wIBAjBVBgNVHREETjBMghByaWNoLmV4YW1wbGUu
Y29thxAgAQ24AAAAAAAAAAAAAAABgRFhZG1pbkBleGFtcGxlLmNvbYYTaHR0cHM6
Ly9leGFtcGxlLmNvbTAeBgNVHSUEFzAVBggrBgEFBQcDAQYJKwYBBAGGjR8BMB4G
A1UdHgEB/wQUMBKgEDAOggwuZXhhbXBsZS5jb20wDQYJKoZIhvcNAQELBQADggEB
AHAKXzFplqL/4AC3x+/VYbHtk/ILd6+DNYA0B3YLDcp/50J+a6VwCOlOXha8xjPA
IqBzX6FDu4G8aXgqF4XCIutGffPykqzzJoQXqLgjWLqRhdFIAnaDXKo8E1UumOrq
j3DOF8MIrVT7PnOud3NZom/pA+dHTpdnBgVGZ7yUZEpIReBH6Q1srDYih9yqg8mJ
uJ/07/KqSsEZ4k3y1bTjMhZ9FkFpr5QwfwNU1sP++/QRlxWaHmoB1GBQVTDfp5a0
CJ5bG1AP0rQTSPUK0ktrRTMuDfHU5fmATuYjYUWUr/K5jnUCuLWJ9wFoWLNAsHpg
7AVpdOhGsyoNIQMin/6g3CE=
-----END CERTIFICATE-----`
	certRichOut = `Version:          3 (0x02)
Serial number:    631358098983425198860480631379491966654151753651 (0x6e971388c095e77b0c6668a87374ec220b797fb3)
Algorithm ID:     SHA256withRSA
Validity
  Not Before:     20/07/2026 01:51:34 (dd-mm-yyyy hh:mm:ss) (260720015134Z)
  Not After:      17/07/2036 01:51:34 (dd-mm-yyyy hh:mm:ss) (360717015134Z)
Issuer
  C  = US
  CN = rich.example.com
Subject
  C  = US
  CN = rich.example.com
Fingerprints
  MD5:            3b86ecc09440b8d2b23d5b83b3892db9
  SHA1:           ee6df940e50fa536ef168a1e1cd47a3a895b8997
  SHA256:         9ad277a533db10338494ad53f2f1a5a19a15483111311561db16dd3fd3c10b96
Public Key
  Algorithm:      RSA
  Length:         2048 bits
  Modulus:        85:89:0d:0f:0d:6b:29:b9:c4:25:73:b4:9d:03:4b:63:
                  02:b6:b5:33:75:49:ab:d5:6b:e0:d1:89:e1:ec:4c:e4:
                  ba:74:61:87:ef:a3:18:d0:1b:b6:36:74:ef:5c:c6:8d:
                  9f:d4:ea:71:90:e5:fd:44:0d:3f:c7:f6:bd:c3:f7:1e:
                  9e:fd:79:06:91:54:67:41:9d:51:54:67:11:c1:69:1d:
                  4a:28:e5:7d:bc:4f:9c:25:6a:74:4e:0d:b3:f2:58:19:
                  b5:41:9b:db:f3:65:b9:f7:e2:d6:7a:89:46:ff:4f:3c:
                  8f:8b:d0:fa:4b:98:5f:3c:4f:55:67:15:bf:bb:32:95:
                  52:9e:6c:5b:63:3f:aa:e8:f1:bb:60:29:4c:97:48:69:
                  c7:e4:ac:0f:70:c3:b2:43:b3:a1:67:65:20:9b:17:bf:
                  e3:77:fd:db:dd:28:a0:a0:e9:14:2c:c9:b9:5f:3c:52:
                  3d:56:9e:b1:ce:db:1a:91:ac:b7:78:05:2a:09:dd:9b:
                  9f:44:b2:ec:99:5d:03:5d:22:8a:95:1d:79:ce:19:5d:
                  18:1d:79:79:03:02:65:8d:db:bc:63:d7:dd:a6:38:c5:
                  c1:70:6f:38:ab:b8:5a:9f:9d:60:39:c5:74:b4:e3:b5:
                  4e:b9:99:48:6e:c1:bc:0b:98:b4:1b:e4:01:22:87:a1
  Exponent:       65537 (0x10001)
Certificate Signature
  Algorithm:      SHA256withRSA
  Signature:      70:0a:5f:31:69:96:a2:ff:e0:00:b7:c7:ef:d5:61:b1:
                  ed:93:f2:0b:77:af:83:35:80:34:07:76:0b:0d:ca:7f:
                  e7:42:7e:6b:a5:70:08:e9:4e:5e:16:bc:c6:33:c0:22:
                  a0:73:5f:a1:43:bb:81:bc:69:78:2a:17:85:c2:22:eb:
                  46:7d:f3:f2:92:ac:f3:26:84:17:a8:b8:23:58:ba:91:
                  85:d1:48:02:76:83:5c:aa:3c:13:55:2e:98:ea:ea:8f:
                  70:ce:17:c3:08:ad:54:fb:3e:73:ae:77:73:59:a2:6f:
                  e9:03:e7:47:4e:97:67:06:05:46:67:bc:94:64:4a:48:
                  45:e0:47:e9:0d:6c:ac:36:22:87:dc:aa:83:c9:89:b8:
                  9f:f4:ef:f2:aa:4a:c1:19:e2:4d:f2:d5:b4:e3:32:16:
                  7d:16:41:69:af:94:30:7f:03:54:d6:c3:fe:fb:f4:11:
                  97:15:9a:1e:6a:01:d4:60:50:55:30:df:a7:96:b4:08:
                  9e:5b:1b:50:0f:d2:b4:13:48:f5:0a:d2:4b:6b:45:33:
                  2e:0d:f1:d4:e5:f9:80:4e:e6:23:61:45:94:af:f2:b9:
                  8e:75:02:b8:b5:89:f7:01:68:58:b3:40:b0:7a:60:ec:
                  05:69:74:e8:46:b3:2a:0d:21:03:22:9f:fe:a0:dc:21

Extensions
  subjectKeyIdentifier :
    76a07e7e755a7b7a3f759626e19612d432fad93d
  authorityKeyIdentifier :
    kid=76a07e7e755a7b7a3f759626e19612d432fad93d
  basicConstraints CRITICAL:
    cA=true, pathLen=2
  subjectAltName :
    dns: rich.example.com
    ip: 2001:db8::1
    rfc822: admin@example.com
    uri: https://example.com
  extKeyUsage :
    serverAuth, 1.3.6.1.4.1.99999.1
  nameConstraints CRITICAL:
`
)

func TestParseX509Certificate(t *testing.T) {
	one := func(name, in, want string) opCase {
		return opCase{name, in, want, core.Recipe{{Op: "Parse X.509 certificate", Args: []any{"PEM"}}}}
	}
	runCases(t, []opCase{
		one("RSA with SAN/BC/KU/EKU", certRSAPEM, certRSAOut),
		one("EC P-256", certECPEM, certECOut),
		one("CA with pathlen/AIA/CRL/policies", certCAPEM, certCAOut),
		one("CA IPv6 SAN, unknown EKU and extension", certRichPEM, certRichOut),
	})
}

// TestParseX509CertificateEdge covers the empty-input, alternate-format and
// malformed-input paths.
func TestParseX509CertificateEdge(t *testing.T) {
	if got, err := runOp(t, "Parse X.509 certificate", "", "PEM"); err != nil || got != "No input" {
		t.Fatalf("No input: got %q err %v", got, err)
	}
	// DER Hex format matches PEM.
	der, err := base64.StdEncoding.DecodeString(pemBody(certRSAPEM))
	if err != nil {
		t.Fatal(err)
	}
	want, err := runOp(t, "Parse X.509 certificate", certRSAPEM, "PEM")
	if err != nil {
		t.Fatal(err)
	}
	got, err := runOp(t, "Parse X.509 certificate", hex.EncodeToString(der), "DER Hex")
	if err != nil || got != want {
		t.Fatalf("DER Hex differs from PEM (err %v)", err)
	}
	for _, c := range []struct{ name, input string }{
		{"bad PEM", "not a pem"},
		{"truncated DER", "3003020101"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := runOp(t, "Parse X.509 certificate", c.input, "DER Hex"); err == nil {
				t.Fatalf("expected an error")
			}
		})
	}
}

func TestFormatCertPublicKeyKinds(t *testing.T) {
	rsa := spki(derTLV("30", derTLV("06", oidRSA)+"0500"),
		derTLV("30", derTLV("02", "00ab")+derTLV("02", "010001")))
	ec := spki(derTLV("30", derTLV("06", oidECPub)+derTLV("06", oidP256)), "0400")
	unknown := spki(derTLV("30", derTLV("06", "2a0304")+"0500"), "00")
	for _, c := range []struct{ name, hexIn, want string }{
		{"RSA", rsa, "Algorithm:      RSA"},
		{"EC", ec, "Algorithm:      EC"},
		{"unknown", unknown, "Unknown Public Key type"},
	} {
		if got, err := formatCertPublicKey(c.hexIn, 0); err != nil || !contains(got, c.want) {
			t.Fatalf("%s: got %q err %v", c.name, got, err)
		}
	}
}

func TestFormatCertDateBranches(t *testing.T) {
	if got := formatCertDate("500101000000Z"); got != "01/01/1950 00:00:00" { // UTCTime, 19xx
		t.Fatalf("19xx: %q", got)
	}
	if got := formatCertDate("short"); got != "short" { // too short => passthrough
		t.Fatalf("short: %q", got)
	}
}

func TestFormatCertSignatureBreakout(t *testing.T) {
	sig := derTLV("30", derTLV("02", "01")+derTLV("02", "02"))
	if !contains(formatCertSignature(sig), "r:") || !contains(formatCertSignature("abcd"), "Signature:") {
		t.Fatal("signature breakout")
	}
}

func TestCertExtensionBodies(t *testing.T) {
	gnDn := derTLV("a4", derName("x"))
	gnOther := derTLV("a0", derTLV("06", "2a0304")+derTLV("a0", derTLV("0c", hx("v"))))
	san := derTLV("30", derTLV("82", hx("e.com"))+derTLV("87", "7f000001")+gnDn+gnOther+
		derTLV("81", hx("a@e"))+derTLV("86", hx("http://e")))
	crldp := derTLV("30", derTLV("30", derTLV("a0", derTLV("a0", derTLV("86", hx("http://crl"))))))
	aia := derTLV("30", derTLV("30", derTLV("06", "2b06010505073002")+derTLV("86", hx("http://ca")))+
		derTLV("30", derTLV("06", "2b06010505073001")+derTLV("86", hx("http://ocsp"))))
	pol := derTLV("30", derTLV("30", derTLV("06", "2a0304")))

	sanOut, _ := certExtSubjectAltName(san)
	crlOut, _ := certExtCRLDistPoints(crldp)
	aiaOut, _ := certExtAuthorityInfoAccess(aia)
	polOut, _ := certExtCertificatePolicies(pol)
	for _, c := range []struct{ name, got, want string }{
		{"basicConstraints {}", certExtBasicConstraints(derTLV("30", "")), "    {}\n"},
		{"basicConstraints cA", certExtBasicConstraints(derTLV("30", "0101ff")), "    cA=true\n"},
		{"basicConstraints pathLen", certExtBasicConstraints(derTLV("30", "0101ff"+derTLV("02", "02"))), "    cA=true, pathLen=2\n"},
		{"authorityKeyID", certExtAuthorityKeyID(derTLV("30", derTLV("80", "abcd"))), "    kid=abcd\n"},
		{"authorityKeyID none", certExtAuthorityKeyID(derTLV("30", "")), ""},
		{"subjectAltName dn", sanOut, "    dn: /CN=x\n"},
		{"subjectAltName other", sanOut, "    other: 1.2.3.4=v\n"},
		{"crlDistPoints", crlOut, "    http://crl\n"},
		{"aia caissuer", aiaOut, "    caissuer: http://ca\n"},
		{"aia ocsp", aiaOut, "    ocsp: http://ocsp\n"},
		{"certPolicies", polOut, "    policy oid: 1.2.3.4\n"},
	} {
		if !contains(c.got, c.want) {
			t.Errorf("%s: got %q want substring %q", c.name, c.got, c.want)
		}
	}
	// keyUsage identifiers from a crafted BIT STRING (digitalSignature + keyEncipherment).
	if ku := keyUsageIdentifiers(derTLV("03", "05a0")); strings.Join(ku, ",") != "digitalSignature,keyEncipherment" {
		t.Fatalf("keyUsage: %v", ku)
	}
	// Unknown extension name yields no body; malformed extensions error.
	if body, _ := formatCertExtensionBody("nameConstraints", "00"); body != "" {
		t.Fatalf("unknown ext body: %q", body)
	}
	if _, err := formatCertificate(bad); err == nil {
		t.Fatal("formatCertificate should error on malformed input")
	}
	if _, err := formatCertExtensions(bad, 0); err == nil {
		t.Fatal("formatCertExtensions should error")
	}
	if _, _, err := parseValidity(bad, 0); err == nil {
		t.Fatal("parseValidity should error")
	}
}

// mkCertHex assembles a Certificate from tbsCertificate children, with a fixed
// signatureAlgorithm and empty signatureValue.
func mkCertHex(tbsKids ...string) string {
	alg := derTLV("30", derTLV("06", "2a864886f70d01010b")+"0500")
	return derTLV("30", derTLV("30", strings.Join(tbsKids, ""))+alg+derTLV("03", "00"))
}

func TestParseCertDepthGuards(t *testing.T) {
	a0v := derTLV("a0", derTLV("02", "02"))
	serial := derTLV("02", "10")
	alg := derTLV("30", derTLV("06", "2a864886f70d01010b")+"0500")
	badAlg := derTLV("30", "")
	dn := derName("x")
	badDN := derTLV("30", derTLV("31", derTLV("30", derTLV("06", "5504"))))
	t1 := derTLV("17", hx("240101000000Z"))
	validity := derTLV("30", t1+t1)
	badValidity := derTLV("30", t1)
	rsa := spki(derTLV("30", derTLV("06", oidRSA)+"0500"),
		derTLV("30", derTLV("02", "00ab")+derTLV("02", "010001")))
	badSPKI := derTLV("30", derTLV("30", "")+derTLV("03", "0000"))

	for _, c := range []struct{ name, hex string }{
		{"tbs<6", mkCertHex(a0v, serial, alg)},
		{"tbs<off+7", mkCertHex(a0v, serial, alg, dn, validity, dn)},
		{"bad sigalg", mkCertHex(a0v, serial, badAlg, dn, validity, dn, rsa)},
		{"bad issuer", mkCertHex(a0v, serial, alg, badDN, validity, dn, rsa)},
		{"bad validity", mkCertHex(a0v, serial, alg, dn, badValidity, dn, rsa)},
		{"bad subject", mkCertHex(a0v, serial, alg, dn, validity, badDN, rsa)},
		{"bad spki", mkCertHex(a0v, serial, alg, dn, validity, dn, badSPKI)},
		{"empty version", mkCertHex(derTLV("a0", ""), serial, alg, dn, validity, dn, rsa)},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := formatCertificate(c.hex); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestParseCertLeafGuards(t *testing.T) {
	errFns := map[string]func() error{
		"formatCertPublicKey":     func() error { _, e := formatCertPublicKey(bad, 0); return e },
		"certExtSubjectAltName":   func() error { _, e := certExtSubjectAltName(bad); return e },
		"certExtCRLDistPoints":    func() error { _, e := certExtCRLDistPoints(bad); return e },
		"certExtAuthorityInfoAcc": func() error { _, e := certExtAuthorityInfoAccess(bad); return e },
		"certExtCertificatePol":   func() error { _, e := certExtCertificatePolicies(bad); return e },
		"extKeyUsageOIDs":         func() error { _, e := extKeyUsageOIDs(bad); return e },
		"formatCertExtBody":       func() error { _, e := formatCertExtensionBody("extKeyUsage", bad); return e },
	}
	for name, fn := range errFns {
		if fn() == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	// No extensions => empty; unknown extension OID => header only (name == OID).
	if got, _ := formatCertExtensions("", -1); got != "" {
		t.Fatalf("no ext: %q", got)
	}
	unknownExt := derTLV("30", derTLV("30", derTLV("06", "2a0304")+derTLV("04", "00")))
	if got, _ := formatCertExtensions(unknownExt, 0); !contains(got, "1.2.3.4 :") {
		t.Fatalf("unknown ext: %q", got)
	}
	// keyUsage with a too-short BIT STRING value yields no identifiers.
	if len(keyUsageIdentifiers(derTLV("03", ""))) != 0 {
		t.Fatal("keyUsage short")
	}
	// authorityKeyIdentifier with no [0] keyid yields empty.
	if certExtAuthorityKeyID(derTLV("30", derTLV("82", "10"))) != "" {
		t.Fatal("aki no keyid")
	}
}

// mkCertHex2 assembles a Certificate with an explicit outer signatureAlgorithm.
func mkCertHex2(rootAlg string, tbsKids ...string) string {
	return derTLV("30", derTLV("30", strings.Join(tbsKids, ""))+rootAlg+derTLV("03", "00"))
}

func TestParseCertDeepGuards(t *testing.T) {
	a0v := derTLV("a0", derTLV("02", "02"))
	serial := derTLV("02", "10")
	alg := derTLV("30", derTLV("06", "2a864886f70d01010b")+"0500")
	badAlg := derTLV("30", "")
	dn := derName("x")
	badDN := derTLV("30", derTLV("31", derTLV("30", derTLV("06", "5504"))))
	t1 := derTLV("17", hx("240101000000Z"))
	validity := derTLV("30", t1+t1)
	badValidity := derTLV("30", t1)
	rsa := spki(derTLV("30", derTLV("06", oidRSA)+"0500"),
		derTLV("30", derTLV("02", "00ab")+derTLV("02", "010001")))
	badSPKI := derTLV("30", derTLV("30", "")+derTLV("03", "0000"))
	extOK := derTLV("a3", derTLV("30", ""))
	// An extensions [3] wrapping an Extension SEQUENCE with a single child.
	badExt := derTLV("a3", derTLV("30", derTLV("30", derTLV("06", "551d0e"))))

	full := func(kids ...string) string { return mkCertHex2(alg, kids...) }
	for _, c := range []struct{ name, hex string }{
		{"bad sigalg", full(a0v, serial, badAlg, dn, validity, dn, rsa, extOK)},
		{"bad issuer", full(a0v, serial, alg, badDN, validity, dn, rsa, extOK)},
		{"bad validity kids", full(a0v, serial, alg, dn, badValidity, dn, rsa, extOK)},
		{"bad subject", full(a0v, serial, alg, dn, validity, badDN, rsa, extOK)},
		{"bad spki", full(a0v, serial, alg, dn, validity, dn, badSPKI, extOK)},
		{"bad extensions", full(a0v, serial, alg, dn, validity, dn, rsa, badExt)},
		{"bad root sigalg", mkCertHex2(badAlg, a0v, serial, alg, dn, validity, dn, rsa, extOK)},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := formatCertificate(c.hex); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestFormatCertPublicKeyGuards(t *testing.T) {
	// SubjectPublicKeyInfo with only one child, empty algorithm, and unknown EC curve.
	if _, err := formatCertPublicKey(derTLV("30", derTLV("30", "")), 0); err == nil {
		t.Error("spki<2")
	}
	if _, err := formatCertPublicKey(derTLV("30", derTLV("30", "")+derTLV("03", "0000")), 0); err == nil {
		t.Error("empty algId")
	}
	// RSA with a malformed public-key bit string.
	rsaBad := spki(derTLV("30", derTLV("06", oidRSA)+"0500"), "300a")
	if _, err := formatCertPublicKey(rsaBad, 0); err == nil {
		t.Error("rsa bad bitstring")
	}
	// Certificate-extension sub-formatter error path.
	if _, err := formatCertExtensionBody("subjectAltName", bad); err == nil {
		t.Error("san body err")
	}
	// A CRL distribution point with an empty fullName is skipped (continue).
	if got, _ := certExtCRLDistPoints(derTLV("30", derTLV("30", derTLV("a0", "")))); got != "" {
		t.Errorf("crldp skip: %q", got)
	}
	// An authorityInfoAccess AccessDescription with a single child is skipped.
	if got, _ := certExtAuthorityInfoAccess(derTLV("30", derTLV("30", derTLV("06", "2b06010505073001")))); got != "" {
		t.Errorf("aia skip: %q", got)
	}
}

func TestParseCertFinalGuards(t *testing.T) {
	// Run: a PEM-format input that fails to decode reaches the format-error path.
	if _, err := runOp(t, "Parse X.509 certificate", "not a pem", "PEM"); err == nil {
		t.Error("bad PEM should error")
	}
	// An extension whose body formatter errors propagates out of formatCertExtensions.
	sanExt := derTLV("30", derTLV("30", derTLV("06", "551d11")+derTLV("04", "300a")))
	if _, err := formatCertExtensions(sanExt, 0); err == nil {
		t.Error("ext body error should propagate")
	}
	// Malformed values fall back to defaults in these helpers.
	if certExtBasicConstraints(bad) != "    {}\n" {
		t.Error("basicConstraints fallback")
	}
	if certExtAuthorityKeyID(bad) != "" {
		t.Error("authorityKeyID fallback")
	}
	// A certificatePolicies entry with no children is skipped.
	if got, _ := certExtCertificatePolicies(derTLV("30", derTLV("30", ""))); got != "" {
		t.Errorf("empty policy skip: %q", got)
	}
}

func TestParseCertNoVersion(t *testing.T) {
	// A v1 certificate omits the [0] version wrapper.
	serial := derTLV("02", "10")
	alg := derTLV("30", derTLV("06", "2a864886f70d01010b")+"0500")
	dn := derName("x")
	t1 := derTLV("17", hx("240101000000Z"))
	validity := derTLV("30", t1+t1)
	rsa := spki(derTLV("30", derTLV("06", oidRSA)+"0500"),
		derTLV("30", derTLV("02", "00ab")+derTLV("02", "010001")))
	extOK := derTLV("a3", derTLV("30", ""))
	h := mkCertHex2(alg, serial, alg, dn, validity, dn, rsa, extOK)
	out, err := formatCertificate(h)
	if err != nil || !contains(out, "Version:          1 (0x00)") {
		t.Fatalf("v1 cert: err %v", err)
	}
}
