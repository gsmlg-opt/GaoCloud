example.com.      3600  IN  SOA sns.dns.icann.org. noc.dns.icann.org. 1 7200 3600 1209600 3600
example.com.      86400 IN  NS  a.iana-servers.net.
a.example.com.    3600  IN  A   1.1.1.1 
mail.example.com. 3600  IN  MX  20 mx.example.com.
mx.example.com.  3600  IN  A   2.2.2.2
