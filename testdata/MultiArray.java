public class MultiArray {
    public static void main(String[] args) {
        // 2D array with multianewarray
        int[][] matrix = new int[3][4];
        matrix[0][0] = 1;
        matrix[1][2] = 5;
        matrix[2][3] = 9;
        System.out.println(matrix[0][0]);   // 1
        System.out.println(matrix[1][2]);   // 5
        System.out.println(matrix[2][3]);   // 9
        System.out.println(matrix.length);  // 3
        System.out.println(matrix[0].length); // 4

        // manual 2D array (array of arrays)
        int[][] grid = new int[2][];
        grid[0] = new int[]{10, 20};
        grid[1] = new int[]{30, 40, 50};
        System.out.println(grid[0][1]);     // 20
        System.out.println(grid[1][2]);     // 50
    }
}
