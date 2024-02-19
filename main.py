from flask import Flask, jsonify

app = Flask(__name__)

def fibonacci(n):
    if n == 0: 
        return 0
    if n == 1:
        return 1
    return fibonacci(n - 1) + fibonacci(n - 2)

@app.route('/hello/<int:num>', methods=['GET'])
def hello(num):
    if num < 0:
        return jsonify(error="Input must be a non-negative integer."), 400

    fib_result = fibonacci(num)
    return jsonify(input=num, fibonacci_sequence=fib_result)

if __name__ == '__main__':
    app.run(debug=True)
