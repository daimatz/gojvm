public class FloatArith {
    public static void main(String[] args) {
        float a = 3.5f;
        float b = 2.0f;

        System.out.println((int)(a + b));   // 5
        System.out.println((int)(a - b));   // 1
        System.out.println((int)(a * b));   // 7
        System.out.println((int)(a / b));   // 1

        // float to int, int to float
        int n = 100;
        float f = (float) n;
        System.out.println((int) f);        // 100

        // float comparison
        float x = 1.5f;
        float y = 2.5f;
        if (x < y) {
            System.out.println(1);          // 1
        }
        if (x > y) {
            System.out.println(0);
        }

        // Float.toString
        System.out.println(String.valueOf((int)(x * 10.0f))); // 15
    }
}
