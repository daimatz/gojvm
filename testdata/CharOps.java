public class CharOps {
    public static void main(String[] args) {
        // char arithmetic
        char c = 'A';
        System.out.println((int) c);          // 65
        System.out.println((char)(c + 1));    // B
        System.out.println((char)(c + 25));   // Z

        // char array
        char[] chars = {'H', 'e', 'l', 'l', 'o'};
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < chars.length; i++) {
            sb.append(chars[i]);
        }
        System.out.println(sb.toString());    // Hello

        // String to char array
        String s = "World";
        char[] arr = s.toCharArray();
        System.out.println(arr.length);       // 5
        System.out.println(arr[0]);           // W

        // Widening/narrowing conversions
        byte b = 127;
        short sh = 32767;
        System.out.println((int) b);          // 127
        System.out.println((int) sh);         // 32767
        System.out.println((byte) 256);       // 0
        System.out.println((short) 65536);    // 0

        // Character methods
        System.out.println(Character.isLetter('A'));  // true
        System.out.println(Character.isDigit('5'));   // true
        System.out.println(Character.toLowerCase('X')); // x
    }
}
