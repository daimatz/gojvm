public class BitwiseOps {
    public static void main(String[] args) {
        int a = 0b1100; // 12
        int b = 0b1010; // 10

        // AND, OR, XOR
        System.out.println(a & b);  // 8  (1000)
        System.out.println(a | b);  // 14 (1110)
        System.out.println(a ^ b);  // 6  (0110)

        // NOT
        System.out.println(~0);     // -1

        // Shifts
        System.out.println(1 << 10);     // 1024
        System.out.println(-128 >> 2);   // -32 (arithmetic shift)
        System.out.println(-128 >>> 2);  // 1073741792 (logical shift)

        // Long bitwise
        long x = 1L << 40;
        System.out.println(x);           // 1099511627776
        System.out.println(x >> 20);     // 1048576
    }
}
