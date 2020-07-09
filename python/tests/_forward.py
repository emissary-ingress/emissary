import sys
import socket
from threading import Thread


def server(dst_host: str, dst_port: int, src_host: str, src_port: int) -> None:
    listen_sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    listen_sock.bind((src_host, src_port))
    listen_sock.listen(5)

    while True:
        src_sock = listen_sock.accept()[0]
        dst_sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        dst_sock.connect((dst_host, dst_port))
        Thread(target=forward, args=(src_sock, dst_sock), daemon=True).start()
        Thread(target=forward, args=(dst_sock, src_sock), daemon=True).start()


def forward(dst: socket.socket, src: socket.socket) -> None:
    while True:
        data = src.recv(64 * 1024)
        if data:
            dst.sendall(data)
        else:
            try:
                # Close destination first as origin is likely already closed
                dst.shutdown(socket.SHUT_WR)
                src.shutdown(socket.SHUT_RD)
            except OSError:
                pass
            return


if __name__ == "__main__":
    dst_host, dst_port_str, src_host, src_port_str = sys.argv[1:]
    dst_port = int(dst_port_str)
    src_port = int(src_port_str)
    server(dst_host, dst_port, src_host, src_port)
