public class Varargs {
    static int sum(int... nums) {
        int total = 0;
        for (int n : nums) {
            total += n;
        }
        return total;
    }

    static String join(String sep, String... words) {
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < words.length; i++) {
            if (i > 0) sb.append(sep);
            sb.append(words[i]);
        }
        return sb.toString();
    }

    public static void main(String[] args) {
        System.out.println(sum(1, 2, 3));           // 6
        System.out.println(sum(10, 20, 30, 40));    // 100
        System.out.println(join(", ", "a", "b", "c")); // a, b, c
    }
}
