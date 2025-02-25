import unittest
import lambda_function as function

class TestCompact(unittest.TestCase):
    def test_removes_empty_attributes(self):
        attributes = {
            'format': 'mp4',
            'fileSize': 1000,
            'duration': None
        }

        expected = {
            'format': 'mp4',
            'fileSize': 1000
        }

        self.assertEqual(function.compact(attributes), expected)

    def test_does_not_remove_zero_attributes(self):
        attributes = {
            'format': 'mp4',
            'fileSize': 0
        }

        expected = {
            'format': 'mp4',
            'fileSize': 0
        }

        self.assertEqual(function.compact(attributes), expected)

if __name__ == '__main__':
    unittest.main()
