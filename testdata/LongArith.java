public class LongArith {
    public static void main(String[] args) {
        long a = 1000000000L;
        long b = 2000000000L;
        long c = a + b;
        System.out.println(c);       // 3000000000
        System.out.println(a * 3L);  // 3000000000
        System.out.println(c - a);   // 2000000000
    }
}
