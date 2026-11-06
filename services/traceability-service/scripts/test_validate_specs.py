"""Exercise runtime contract validation HTTP transport without real credentials."""
import threading
import unittest
from contextlib import contextmanager
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.error import HTTPError

from validate_specs import read_runtime_profile


@contextmanager
def serve(handler):
    server = ThreadingHTTPServer(('127.0.0.1', 0), handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield 'http://127.0.0.1:' + str(server.server_port)
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)


class RuntimeTransportTest(unittest.TestCase):
    def test_reads_original_authenticated_response(self):
        received = []

        class Endpoint(BaseHTTPRequestHandler):
            def do_GET(self):
                received.append(self.headers.get('Authorization'))
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(b'{"id":"synthetic-profile"}')

            def log_message(self, *args):
                pass

        with serve(Endpoint) as origin:
            self.assertEqual(read_runtime_profile(origin, 'synthetic-test-token'), {'id': 'synthetic-profile'})
        self.assertEqual(received, ['Bearer synthetic-test-token'])

    def test_redirect_cannot_receive_bearer_token(self):
        received = []

        class Receiver(BaseHTTPRequestHandler):
            def do_GET(self):
                received.append(self.headers.get('Authorization'))
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b'{}')

            def log_message(self, *args):
                pass

        with serve(Receiver) as target:
            for status in (301, 302, 303, 307, 308):
                with self.subTest(status=status):
                    class Redirect(BaseHTTPRequestHandler):
                        def do_GET(self):
                            self.send_response(status)
                            self.send_header('Location', target + '/capture')
                            self.end_headers()

                        def log_message(self, *args):
                            pass

                    with serve(Redirect) as origin:
                        with self.assertRaises(HTTPError) as error:
                            read_runtime_profile(origin, 'synthetic-test-token')
                        self.assertEqual(error.exception.code, status)
                        error.exception.close()
        self.assertEqual(received, [])


if __name__ == '__main__':
    unittest.main()
