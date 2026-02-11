public class DoubleArith {
    public static void main(String[] args) {
        double a = 3.5;
        double b = 2.0;
        System.out.println((int)(a + b));   // 5
        System.out.println((int)(a - b));   // 1
        System.out.println((int)(a * b));   // 7
        System.out.println((int)(a / b));   // 1
        // double -> int conversion
        double c = 9.99;
        System.out.println((int)c);         // 9
        // int -> double -> int roundtrip
        int x = 42;
        double d = x;
        System.out.println((int)d);         // 42
    }
}
