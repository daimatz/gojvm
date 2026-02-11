public class Lambda {
    interface MathOp {
        int apply(int a, int b);
    }
    public static void main(String[] args) {
        MathOp add = (a, b) -> a + b;
        MathOp mul = (a, b) -> a * b;
        System.out.println(add.apply(3, 4));
        System.out.println(mul.apply(3, 4));
    }
}
