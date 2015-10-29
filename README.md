Pattern MaTching (pmt)
Rafael Farias Marinheiro - rfm3@cin.ufpe.br

# Compilando

Para compilar o programa, execute os seguintes passos:

1. Instale o [Go](https://golang.org/doc/install) e configure o GOPATH
2. Extraia o código dentro do seu GOPATH. É importante garantir que a estrutura de pastas fique da seguinte forma:


		$GOPATH/
			bin/
			src/
				github.com/
					RafaelMarinheiro
						streammatch/
							kmp.go
							ahocorasick.go
							...
							pmt/
								pmt.go


3. Para dar build, faça

		cd $GOPATH/src/github.com/RafaelMarinheiro/streammatch/pmt && go build pmt.go

	O executável estará no seguinte path 
	
		$GOPATH/src/github.com/RafaelMarinheiro/streammatch/pmt/pmt

4. Para instalar, faça:
	
		go install github.com/RafaelMarinheiro/streammatch/pmt

	O executável estará no seguinte path

		$GOPATH/bin/pmt

# Executando

	Usage: pmt [-hsv] [--cpuprofile path] [-e max_dist] [--memprofile path] [-p filepath] needle [haystack ...]
	 -e, --edit=max_dist
	                Compute the approximate matching
	 -h, --help     Shows this message
	 -p, --pattern=filepath
	                Use line-break separated patterns from a file
	 -s, --simple   Show simple output
	 -v, --verbose  Show log messages
	 needle - only if -p was not used
	 haystack