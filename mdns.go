package main

import "github.com/libp2p/zeroconf/v2"

// startMDNS публикует в локальной сети постоянное имя ipdrop.local,
// указывающее на текущий IP компьютера. Благодаря этому значок на экране
// «Домой» у телефона продолжает работать, даже если IP поменялся.
//
// Возвращает функцию остановки. Если сеть/роутер блокируют mDNS — вернёт ошибку,
// и приложение просто продолжит работать по обычному IP-адресу.
func startMDNS(port int, ip string) (func(), error) {
	server, err := zeroconf.RegisterProxy(
		"IpDrop",                          // имя экземпляра службы
		"_http._tcp",                      // тип службы
		"local.",                          // домен
		port,                              // порт
		stableHost,                        // host -> ipdrop.local
		[]string{ip},                      // адреса для A-записи
		[]string{"IpDrop file receiver"},  // TXT
		nil,                               // все сетевые интерфейсы
	)
	if err != nil {
		return nil, err
	}
	return server.Shutdown, nil
}
